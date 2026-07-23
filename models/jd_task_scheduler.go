package models

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	jdDefaultMaxWorkers        = 200
	jdDefaultMaxUserJobs       = 10
	jdDefaultJobTimeoutMinutes = 60
)

type jdJobState int

const (
	jdJobQueued jdJobState = iota
	jdJobRunning
	jdJobDone
	jdJobCancelled
)

type jdJobSpec struct {
	TaskType   string
	TaskName   string
	PtPin      string
	Nickname   string
	ScriptPath string
	Envs       map[string]string
	Parser     func(string, *Sender) string
}

type JdJob struct {
	ID         string
	TaskLogID  string
	UserID     int
	TaskType   string
	TaskName   string
	PtPin      string
	ScriptPath string
	Envs       map[string]string
	Parser     func(string, *Sender) string
	LogChan    chan string
	Sender     *Sender
	OnComplete func(fullOutput string, runErr error)
	batchLog *jdBatchLogWriter

	state   jdJobState
	slotKey string
	cancel  context.CancelFunc
	cmdMu   sync.Mutex
	cmd     *exec.Cmd

	EnqueuedAt     time.Time
	StartedAt      time.Time
	ClientSource   string
	ClientPlatform string
}

func (j *JdJob) isCancelled() bool {
	j.cmdMu.Lock()
	defer j.cmdMu.Unlock()
	return j.state == jdJobCancelled
}

func (j *JdJob) setCmd(cmd *exec.Cmd) {
	j.cmdMu.Lock()
	defer j.cmdMu.Unlock()
	j.cmd = cmd
}

func (j *JdJob) getCmd() *exec.Cmd {
	j.cmdMu.Lock()
	defer j.cmdMu.Unlock()
	return j.cmd
}

func (j *JdJob) markCancelled() {
	j.cmdMu.Lock()
	defer j.cmdMu.Unlock()
	if j.state == jdJobDone || j.state == jdJobCancelled {
		return
	}
	j.state = jdJobCancelled
	if j.cancel != nil {
		j.cancel()
	}
	if j.cmd != nil && j.cmd.Process != nil {
		killProcessGroup(j.cmd)
	}
}

type jdPortalBatch struct {
	taskLogID    string
	portalTaskID string
	userID       int
	remaining    int32
	logChan      chan string
	runRecordID  int64
	batchLog     *jdBatchLogWriter
}

type jdBatchLogWriter struct {
	mu     sync.Mutex
	file   *os.File
	name   string
	closed bool
}

func (w *jdBatchLogWriter) writeAccountHeader(job *JdJob) {
	if w == nil || job == nil {
		return
	}
	w.writeRaw(fmt.Sprintf("\n===== %s | %s | %s =====\n",
		time.Now().Format("2006-01-02 15:04:05"), job.TaskName, job.PtPin))
}

func (w *jdBatchLogWriter) writeAccountLine(ptPin, stream, line string) {
	if w == nil || strings.TrimSpace(line) == "" {
		return
	}
	prefix := fmt.Sprintf("[%s]", ptPin)
	if stream != "" {
		prefix += fmt.Sprintf("[%s]", stream)
	}
	w.writeRaw(fmt.Sprintf("%s %s\n", prefix, line))
}

func (w *jdBatchLogWriter) writeRaw(content string) {
	if w == nil || content == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.file == nil {
		return
	}
	_, _ = w.file.WriteString(content)
}

func (w *jdBatchLogWriter) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	if w.file != nil {
		_, _ = w.file.WriteString(fmt.Sprintf("\n===== 批次结束 %s =====\n", time.Now().Format("2006-01-02 15:04:05")))
		_ = w.file.Close()
	}
}

type JdTaskScheduler struct {
	mu            sync.Mutex
	queue         []*JdJob
	queueCond     *sync.Cond
	jobsByID      map[string]*JdJob
	batches       map[string]*jdPortalBatch
	taskLogJobs   map[string]map[string]struct{}
	userJobCounts map[int]int
	slots         map[string]string

	workerMu     sync.Mutex
	workerCond   *sync.Cond
	workerUsed   int
	maxWorkers   int
	maxUserJobs  int
	jobTimeout   time.Duration
	started      bool
}

var (
	jdScheduler     *JdTaskScheduler
	jdSchedulerOnce sync.Once
)

func jdResolvedMaxWorkers() int {
	if Config.JdTask.MaxWorkers > 0 {
		return Config.JdTask.MaxWorkers
	}
	return jdDefaultMaxWorkers
}

func jdResolvedMaxUserJobs() int {
	if Config.JdTask.MaxUserJobs > 0 {
		return Config.JdTask.MaxUserJobs
	}
	return jdDefaultMaxUserJobs
}

func jdResolvedJobTimeout() time.Duration {
	mins := Config.JdTask.JobTimeoutMinutes
	if mins <= 0 {
		mins = jdDefaultJobTimeoutMinutes
	}
	return time.Duration(mins) * time.Minute
}

func InitJdTaskScheduler() {
	jdSchedulerOnce.Do(func() {
		cleanupOrphanJdNodeProcesses()
		jdScheduler = newJdTaskScheduler()
		jdScheduler.start()
		RegisterConfigReloadHook(func() {
			if jdScheduler != nil {
				jdScheduler.ApplyRuntimeConfig()
			}
		})
	})
}

func GetJdTaskScheduler() *JdTaskScheduler {
	InitJdTaskScheduler()
	return jdScheduler
}

func newJdTaskScheduler() *JdTaskScheduler {
	s := &JdTaskScheduler{
		jobsByID:      make(map[string]*JdJob),
		batches:       make(map[string]*jdPortalBatch),
		taskLogJobs:   make(map[string]map[string]struct{}),
		userJobCounts: make(map[int]int),
		slots:         make(map[string]string),
		maxWorkers:    jdResolvedMaxWorkers(),
		maxUserJobs:   jdResolvedMaxUserJobs(),
		jobTimeout:    jdResolvedJobTimeout(),
	}
	s.queueCond = sync.NewCond(&s.mu)
	s.workerCond = sync.NewCond(&s.workerMu)
	return s
}

func (s *JdTaskScheduler) start() {
	if s.started {
		return
	}
	s.started = true
	go s.dispatchLoop()
}

// ApplyRuntimeConfig 热更新 Worker / 每用户上限 / 超时
func (s *JdTaskScheduler) ApplyRuntimeConfig() {
	maxWorkers := jdResolvedMaxWorkers()
	maxUserJobs := jdResolvedMaxUserJobs()
	timeout := jdResolvedJobTimeout()

	s.mu.Lock()
	s.maxUserJobs = maxUserJobs
	s.jobTimeout = timeout
	s.mu.Unlock()

	s.workerMu.Lock()
	old := s.maxWorkers
	s.maxWorkers = maxWorkers
	if maxWorkers > old {
		s.workerCond.Broadcast()
	}
	s.workerMu.Unlock()
	Info("京东任务调度配置已生效: workers=%d userJobs=%d timeout=%v", maxWorkers, maxUserJobs, timeout)
}

func (s *JdTaskScheduler) acquireWorker() {
	s.workerMu.Lock()
	for s.workerUsed >= s.maxWorkers {
		s.workerCond.Wait()
	}
	s.workerUsed++
	s.workerMu.Unlock()
}

func (s *JdTaskScheduler) releaseWorker() {
	s.workerMu.Lock()
	if s.workerUsed > 0 {
		s.workerUsed--
	}
	s.workerCond.Signal()
	s.workerMu.Unlock()
}

func (s *JdTaskScheduler) currentLimits() (maxWorkers, maxUserJobs int, timeout time.Duration) {
	s.mu.Lock()
	maxUserJobs = s.maxUserJobs
	timeout = s.jobTimeout
	s.mu.Unlock()
	s.workerMu.Lock()
	maxWorkers = s.maxWorkers
	s.workerMu.Unlock()
	return
}

func (s *JdTaskScheduler) dispatchLoop() {
	for {
		job := s.dequeue()
		if job == nil {
			continue
		}
		if job.isCancelled() {
			s.releaseJob(job)
			s.finishPortalBatch(job)
			continue
		}
		s.acquireWorker()
		go func(j *JdJob) {
			defer s.releaseWorker()
			s.executeJob(j)
		}(job)
	}
}

func (s *JdTaskScheduler) dequeue() *JdJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.queue) == 0 {
		s.queueCond.Wait()
	}
	job := s.queue[0]
	s.queue = s.queue[1:]
	if job.state == jdJobQueued {
		job.state = jdJobRunning
	}
	return job
}

func jdSlotKey(userID int, taskType, ptPin string) string {
	return fmt.Sprintf("%d:%s:%s", userID, normalizeJdTaskType(taskType), ptPin)
}

func normalizeJdTaskType(taskType string) string {
	return strings.TrimSpace(taskType)
}

func jdTaskTypeFromScript(scriptPath string) string {
	base := strings.TrimSuffix(filepath.Base(scriptPath), ".js")
	base = strings.TrimPrefix(base, "jd_")
	if base == "" {
		return filepath.Base(scriptPath)
	}
	return base
}

// HasUserTask 同一用户同一任务是否已有排队/执行中的 job
func (s *JdTaskScheduler) HasUserTask(userID int, taskType string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := fmt.Sprintf("%d:%s:", userID, normalizeJdTaskType(taskType))
	for slot := range s.slots {
		if strings.HasPrefix(slot, prefix) {
			return true
		}
	}
	return false
}

func jdPinFromEnvs(envs map[string]string) string {
	pins := strings.TrimSpace(envs["pins"])
	return strings.TrimPrefix(pins, "&")
}

func copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func newJdJobID() string {
	return fmt.Sprintf("jdjob_%d", time.Now().UnixNano())
}

func (s *JdTaskScheduler) releaseJob(job *JdJob) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job.slotKey != "" {
		if owner, ok := s.slots[job.slotKey]; ok && owner == job.ID {
			delete(s.slots, job.slotKey)
		}
	}
	if s.userJobCounts[job.UserID] > 0 {
		s.userJobCounts[job.UserID]--
	}
	delete(s.jobsByID, job.ID)
	if job.TaskLogID != "" {
		if ids, ok := s.taskLogJobs[job.TaskLogID]; ok {
			delete(ids, job.ID)
			if len(ids) == 0 {
				delete(s.taskLogJobs, job.TaskLogID)
			}
		}
	}
}

func (s *JdTaskScheduler) submitSpecs(userID int, taskLogID string, logChan chan string, sender *Sender, specs []jdJobSpec, onComplete func(fullOutput string, runErr error), clientCtx ClientContext, runRecordID int64, portalTaskID string) error {
	if len(specs) == 0 {
		return fmt.Errorf("没有可执行的任务")
	}

	s.mu.Lock()
	current := s.userJobCounts[userID]
	limit := s.maxUserJobs
	if limit <= 0 {
		limit = jdDefaultMaxUserJobs
	}
	if current+len(specs) > limit {
		s.mu.Unlock()
		return fmt.Errorf("同一用户同时最多 %d 个任务，当前已有 %d 个", limit, current)
	}
	taskType := normalizeJdTaskType(specs[0].TaskType)
	_ = taskType
	for _, spec := range specs {
		slot := jdSlotKey(userID, spec.TaskType, spec.PtPin)
		if _, ok := s.slots[slot]; ok {
			s.mu.Unlock()
			return fmt.Errorf("账号 %s 的任务 %s 正在执行或排队中", spec.PtPin, normalizeJdTaskType(spec.TaskType))
		}
	}

	var batch *jdPortalBatch
	var batchLog *jdBatchLogWriter
	if taskLogID != "" {
		batchLog = openJdTaskBatchLog(userID, runRecordID, specs[0].TaskType)
		batch = &jdPortalBatch{
			taskLogID:    taskLogID,
			portalTaskID: portalTaskID,
			userID:       userID,
			remaining:    int32(len(specs)),
			logChan:      logChan,
			runRecordID:  runRecordID,
			batchLog:     batchLog,
		}
		s.batches[taskLogID] = batch
	}

	jobs := make([]*JdJob, 0, len(specs))
	for _, spec := range specs {
		job := &JdJob{
			ID:             newJdJobID(),
			TaskLogID:      taskLogID,
			UserID:         userID,
			TaskType:       spec.TaskType,
			TaskName:       spec.TaskName,
			PtPin:          spec.PtPin,
			ScriptPath:     spec.ScriptPath,
			Envs:           copyStringMap(spec.Envs),
			Parser:         spec.Parser,
			LogChan:        logChan,
			Sender:         sender,
			OnComplete:     onComplete,
			state:          jdJobQueued,
			slotKey:        jdSlotKey(userID, spec.TaskType, spec.PtPin),
			EnqueuedAt:     time.Now(),
			ClientSource:   clientCtx.Source,
			ClientPlatform: clientCtx.Platform,
			batchLog:       batchLog,
		}
		s.jobsByID[job.ID] = job
		s.slots[job.slotKey] = job.ID
		s.userJobCounts[userID]++
		if taskLogID != "" {
			if s.taskLogJobs[taskLogID] == nil {
				s.taskLogJobs[taskLogID] = make(map[string]struct{})
			}
			s.taskLogJobs[taskLogID][job.ID] = struct{}{}
		}
		jobs = append(jobs, job)
	}

	basePos := len(s.queue)
	for i, job := range jobs {
		s.queue = append(s.queue, job)
		if logChan != nil && specs[i].Nickname != "" {
			safeLogSend(logChan, fmt.Sprintf("提交账号任务: %s (%s)", specs[i].Nickname, specs[i].PtPin))
		}
	}
	s.queueCond.Signal()
	s.mu.Unlock()

	for i, job := range jobs {
		if job.LogChan != nil {
			safeLogSend(job.LogChan, fmt.Sprintf("已入队，前方还有 %d 个任务等待", basePos+i))
		}
	}
	return nil
}

func (s *JdTaskScheduler) submitPortalBatch(userID int, portalTaskID, taskLogID string, logChan chan string, specs []jdJobSpec, clientCtx ClientContext, runRecordID int64) error {
	return s.submitSpecs(userID, taskLogID, logChan, nil, specs, nil, clientCtx, runRecordID, portalTaskID)
}

func (s *JdTaskScheduler) executeJob(job *JdJob) {
	if job.isCancelled() {
		s.releaseJob(job)
		s.finishPortalBatch(job)
		return
	}

	if job.LogChan != nil {
		safeLogSend(job.LogChan, fmt.Sprintf("开始执行: %s (%s)", job.TaskName, job.PtPin))
	}
	job.cmdMu.Lock()
	if job.StartedAt.IsZero() {
		job.StartedAt = time.Now()
	}
	job.cmdMu.Unlock()
	JD().Infof("JD任务开始 user=%d task=%s pin=%s script=%s", job.UserID, job.TaskType, job.PtPin, job.ScriptPath)

	fullOutput, runErr := s.runNodeJob(job)

	if job.isCancelled() {
		runErr = fmt.Errorf("任务已取消")
	} else if runErr != nil {
		JD().Warnf("JD任务异常 user=%d task=%s pin=%s err=%v", job.UserID, job.TaskType, job.PtPin, runErr)
	}

	if job.LogChan != nil {
		if runErr != nil {
			safeLogSend(job.LogChan, fmt.Sprintf("错误: %v", runErr))
		}
		if job.Parser != nil {
			sender := &Sender{}
			result := job.Parser(fullOutput, sender)
			safeLogSend(job.LogChan, "===== 任务结果 =====")
			safeLogSend(job.LogChan, result)
		}
	}

	if job.OnComplete != nil {
		job.OnComplete(fullOutput, runErr)
	} else if job.Sender != nil && job.Parser != nil && !job.isCancelled() {
		if runErr != nil && strings.TrimSpace(fullOutput) == "" {
			job.Sender.Reply(fmt.Sprintf("%s任务失败：%v", job.TaskName, runErr))
		} else {
			job.Sender.Reply(job.Parser(fullOutput, job.Sender))
		}
	}

	job.cmdMu.Lock()
	job.state = jdJobDone
	job.cmdMu.Unlock()

	s.releaseJob(job)
	s.finishPortalBatch(job)
}

func (s *JdTaskScheduler) finishPortalBatch(job *JdJob) {
	if job.TaskLogID == "" {
		return
	}
	s.mu.Lock()
	batch, ok := s.batches[job.TaskLogID]
	if !ok {
		s.mu.Unlock()
		return
	}
	remaining := atomic.AddInt32(&batch.remaining, -1)
	s.mu.Unlock()

	if remaining > 0 {
		return
	}

	s.mu.Lock()
	delete(s.batches, job.TaskLogID)
	s.mu.Unlock()
	batch.batchLog.close()

	if batch.logChan != nil {
		safeLogSend(batch.logChan, "=====DONE=====所有账号任务执行完成")
	}
	if batch.runRecordID > 0 {
		FinishPortalJdRunRecord(batch.runRecordID, "success", "执行完成")
	}
	RemoveTaskLogChannel(job.TaskLogID)
}

func (s *JdTaskScheduler) runNodeJob(job *JdJob) (string, error) {
	if _, err := os.Stat(job.ScriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("脚本文件不存在 %s", job.ScriptPath)
	}

	envs := copyStringMap(job.Envs)
	if job.TaskLogID != "" && job.Sender == nil {
		ApplyPortalJdTaskProxyEnvs(job.UserID, envs)
	} else {
		ApplyManualJdTaskProxyEnvs(envs)
	}

	timeout := s.jobTimeout
	if timeout <= 0 {
		timeout = jdResolvedJobTimeout()
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	job.cancel = cancel
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", job.ScriptPath)
	cmd.Dir = filepath.Dir(job.ScriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = os.Environ()
	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	job.batchLog.writeAccountHeader(job)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("获取输出管道失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("获取错误管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动脚本失败: %w", err)
	}
	job.setCmd(cmd)

	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err2 := reader.ReadString('\n')
			if len(strings.TrimSpace(line)) > 0 {
				trimmed := strings.TrimSpace(line)
				JD().Infof("[%s] stderr: %s", job.TaskName, trimmed)
				job.batchLog.writeAccountLine(job.PtPin, "stderr", trimmed)
				if job.LogChan != nil {
					safeLogSend(job.LogChan, fmt.Sprintf("[stderr] %s", trimmed))
				}
			}
			if err2 != nil {
				if err2 != io.EOF && len(strings.TrimSpace(line)) > 0 {
					continue
				}
				break
			}
		}
	}()

	var fullOutput strings.Builder
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadString('\n')
		if len(line) > 0 {
			fullOutput.WriteString(line)
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				JD().Infof("[%s] %s", job.TaskName, trimmed)
				job.batchLog.writeAccountLine(job.PtPin, "", trimmed)
				if job.LogChan != nil {
					safeLogSend(job.LogChan, trimmed)
				}
			}
		}
		if err2 != nil {
			break
		}
	}

	waitErr := cmd.Wait()
	output := fullOutput.String()

	if job.isCancelled() {
		return output, fmt.Errorf("任务已取消")
	}
	if ctx.Err() == context.DeadlineExceeded {
		killProcessGroup(cmd)
		return output, fmt.Errorf("任务超时（1小时）")
	}
	if waitErr != nil && strings.TrimSpace(output) == "" {
		return output, waitErr
	}
	return output, nil
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func openJdTaskBatchLog(userID int, runRecordID int64, taskType string) *jdBatchLogWriter {
	dir := jdManualLogDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
	}
	safeTaskType := strings.NewReplacer("/", "_", "\\", "_").Replace(normalizeJdTaskType(taskType))
	if safeTaskType == "" {
		safeTaskType = "jd_task"
	}
	name := fmt.Sprintf("%s_user%d_%d.log", safeTaskType, userID, time.Now().UnixNano())
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		JD().Warnf("[手动京东任务] 创建日志文件失败: %v", err)
		return nil
	}
	writer := &jdBatchLogWriter{file: f, name: name}
	writer.writeRaw(fmt.Sprintf("===== 批次开始 %s =====\n", time.Now().Format("2006-01-02 15:04:05")))
	if runRecordID > 0 {
		SetPortalJdRunLogFile(runRecordID, name)
	}
	return writer
}

func cleanupOrphanJdNodeProcesses() {
	out, err := exec.Command("pgrep", "-f", "自定义执行京东脚本/6dylan6_jdpro").Output()
	if err != nil {
		out, err = exec.Command("pgrep", "-f", "6dylan6_jdpro").Output()
		if err != nil {
			return
		}
	}
	pids := strings.Fields(string(out))
	if len(pids) == 0 {
		return
	}
	for _, pid := range pids {
		_ = exec.Command("kill", "-9", pid).Run()
	}
	JD().Warnf("xdd 启动时已清理 %d 个 JD Node 孤儿进程", len(pids))
}

func (s *JdTaskScheduler) StopTaskLog(taskLogID string) {
	s.mu.Lock()
	jobIDs := make([]string, 0)
	if ids, ok := s.taskLogJobs[taskLogID]; ok {
		for id := range ids {
			jobIDs = append(jobIDs, id)
		}
	}
	s.mu.Unlock()

	for _, id := range jobIDs {
		s.mu.Lock()
		job, ok := s.jobsByID[id]
		s.mu.Unlock()
		if ok {
			job.markCancelled()
		}
	}

	s.mu.Lock()
	newQueue := make([]*JdJob, 0, len(s.queue))
	cancelled := make([]*JdJob, 0)
	for _, job := range s.queue {
		if job.TaskLogID == taskLogID {
			job.markCancelled()
			cancelled = append(cancelled, job)
			continue
		}
		newQueue = append(newQueue, job)
	}
	s.queue = newQueue
	s.mu.Unlock()

	for _, job := range cancelled {
		s.releaseJob(job)
		s.finishPortalBatch(job)
	}
}

// JdAdminTaskView admin 任务列表项
type JdAdminTaskView struct {
	JobID        string `json:"jobId"`
	TaskLogID    string `json:"taskLogId"`
	UserID       int    `json:"userId"`
	TaskType     string `json:"taskType"`
	TaskName     string `json:"taskName"`
	PtPin        string `json:"ptPin"`
	Script       string `json:"script"`
	State        string `json:"state"`
	Source       string `json:"source"`
	Platform     string `json:"platform"`
	SourceLabel  string `json:"sourceLabel"`
	SourceTagCls string `json:"sourceTagCls"`
	PID          int    `json:"pid"`
	QueuePos     int    `json:"queuePos"`
	EnqueuedAt   string `json:"enqueuedAt"`
	StartedAt    string `json:"startedAt"`
	RunningFor   string `json:"runningFor"`
	WaitFor      string `json:"waitFor"`
}

// JdSchedulerStats 调度器统计
type JdSchedulerStats struct {
	MaxWorkers        int `json:"maxWorkers"`
	MaxUserJobs       int `json:"maxUserJobs"`
	Running           int `json:"running"`
	Queued            int `json:"queued"`
	TotalActive       int `json:"totalActive"`
	ActiveUsers       int `json:"activeUsers"`
	JobTimeoutMinutes int `json:"jobTimeoutMinutes"`
}

func jdJobStateLabel(state jdJobState) string {
	switch state {
	case jdJobQueued:
		return "queued"
	case jdJobRunning:
		return "running"
	default:
		return "unknown"
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%dm%ds", sec/60, sec%60)
	}
	return fmt.Sprintf("%dh%dm", sec/3600, (sec%3600)/60)
}

// AdminGetSnapshot 获取调度器快照（admin 页面）
func (s *JdTaskScheduler) AdminGetSnapshot() (JdSchedulerStats, []JdAdminTaskView) {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxUserJobs := s.maxUserJobs
	if maxUserJobs <= 0 {
		maxUserJobs = jdDefaultMaxUserJobs
	}
	timeout := s.jobTimeout
	if timeout <= 0 {
		timeout = time.Duration(jdDefaultJobTimeoutMinutes) * time.Minute
	}
	s.workerMu.Lock()
	maxWorkers := s.maxWorkers
	if maxWorkers <= 0 {
		maxWorkers = jdDefaultMaxWorkers
	}
	s.workerMu.Unlock()
	stats := JdSchedulerStats{
		MaxWorkers:        maxWorkers,
		MaxUserJobs:       maxUserJobs,
		Queued:            len(s.queue),
		JobTimeoutMinutes: int(timeout / time.Minute),
	}

	queuePos := make(map[string]int, len(s.queue))
	for i, job := range s.queue {
		queuePos[job.ID] = i
	}

	activeUsers := make(map[int]struct{})
	views := make([]JdAdminTaskView, 0, len(s.jobsByID))
	now := time.Now()

	for _, job := range s.jobsByID {
		if job.state != jdJobQueued && job.state != jdJobRunning {
			continue
		}
		if job.state == jdJobRunning {
			stats.Running++
		}
		activeUsers[job.UserID] = struct{}{}

		clientCtx := ClientContext{Source: job.ClientSource, Platform: job.ClientPlatform}
		if clientCtx.Source == "" {
			if job.TaskLogID != "" {
				clientCtx = ClientContext{Source: ClientSourceWeb, Platform: ClientPlatformWeb}
			} else {
				clientCtx = BotContext()
			}
		}

		pid := 0
		if cmd := job.getCmd(); cmd != nil && cmd.Process != nil {
			pid = cmd.Process.Pid
		}

		view := JdAdminTaskView{
			JobID:        job.ID,
			TaskLogID:    job.TaskLogID,
			UserID:       job.UserID,
			TaskType:     normalizeJdTaskType(job.TaskType),
			TaskName:     job.TaskName,
			PtPin:        job.PtPin,
			Script:       filepath.Base(job.ScriptPath),
			State:        jdJobStateLabel(job.state),
			Source:       clientCtx.Source,
			Platform:     clientCtx.Platform,
			SourceLabel:  clientCtx.AdminLabel(),
			SourceTagCls: clientCtx.AdminTagClass(),
			PID:          pid,
			QueuePos:     queuePos[job.ID],
			EnqueuedAt:   job.EnqueuedAt.Format("2006-01-02 15:04:05"),
		}
		if !job.StartedAt.IsZero() {
			view.StartedAt = job.StartedAt.Format("2006-01-02 15:04:05")
			view.RunningFor = formatDuration(now.Sub(job.StartedAt))
		} else if job.state == jdJobQueued {
			view.WaitFor = formatDuration(now.Sub(job.EnqueuedAt))
		}
		views = append(views, view)
	}

	stats.TotalActive = len(views)
	stats.ActiveUsers = len(activeUsers)
	stats.Queued = len(s.queue)
	return stats, views
}

// AdminKillJob 管理员强制停止单个 job
func (s *JdTaskScheduler) AdminKillJob(jobID string) error {
	s.mu.Lock()
	job, ok := s.jobsByID[jobID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("任务不存在或已结束")
	}
	wasQueued := job.state == jdJobQueued
	s.mu.Unlock()

	job.markCancelled()

	if wasQueued {
		s.mu.Lock()
		newQueue := make([]*JdJob, 0, len(s.queue))
		for _, q := range s.queue {
			if q.ID == jobID {
				continue
			}
			newQueue = append(newQueue, q)
		}
		s.queue = newQueue
		s.mu.Unlock()
		s.releaseJob(job)
		s.finishPortalBatch(job)
	}
	return nil
}

// AdminKillUserJobs 停止某用户全部排队/运行中的 job
func (s *JdTaskScheduler) AdminKillUserJobs(userID int) int {
	s.mu.Lock()
	ids := make([]string, 0)
	for id, job := range s.jobsByID {
		if job.UserID == userID && (job.state == jdJobQueued || job.state == jdJobRunning) {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()

	for _, id := range ids {
		_ = s.AdminKillJob(id)
	}
	return len(ids)
}

// AdminCleanupOrphanNodes 清理孤儿 node 进程
func AdminCleanupOrphanNodes() int {
	out, err := exec.Command("pgrep", "-f", "6dylan6_jdpro").Output()
	if err != nil {
		return 0
	}
	pids := strings.Fields(string(out))
	for _, pid := range pids {
		_ = exec.Command("kill", "-9", pid).Run()
	}
	if len(pids) > 0 {
		JD().Warnf("管理员手动清理 %d 个 JD Node 进程", len(pids))
	}
	return len(pids)
}
