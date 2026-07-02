package models

import (
//	"encoding/json"
	"fmt"
	"math/rand"
//	"os"
//	"path/filepath"
	"strings"
	"time"

//	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
//  职业定义
// ============================================================

// ClassType 职业类型
type ClassType string

const (
	ClassWarrior  ClassType = "战士"   // 战士：高血量高物防，技能偏物理
	ClassMage     ClassType = "法师"   // 法师：高智力高魔攻，玻璃炮
	ClassAssassin ClassType = "刺客"   // 刺客：高敏捷暴击，爆发极高
	ClassPriest   ClassType = "牧师"   // 牧师：高治疗量，可回血反持久
	ClassPaladin  ClassType = "圣骑士" // 圣骑士：均衡型，攻守兼备
	ClassRanger   ClassType = "游侠"   // 游侠：高运气高闪避，命运流
)

// ClassDef 职业基础数据
type ClassDef struct {
	Name        ClassType
	Emoji       string
	Description string
	HPBonus     int // 额外血量
	StrBonus    int
	IntBonus    int
	AgiBonus    int
	LuckBonus   int
	PDefBonus   int
	MDefBonus   int
	SkillPool   []string // 可学习技能名称
}

var AllClasses = []ClassDef{
	{
		Name: ClassWarrior, Emoji: "⚔️", Description: "厚甲持盾的近战王者，血量最厚，越打越强",
		HPBonus: 80, StrBonus: 30, PDefBonus: 25, AgiBonus: 5, IntBonus: 5, LuckBonus: 5, MDefBonus: 10,
		SkillPool: []string{"重击", "旋风斩", "破甲一击", "铁壁防御", "战嚎激励", "无敌霸体", "复仇之刃"},
	},
	{
		Name: ClassMage, Emoji: "🔮", Description: "智慧之巅的术法大师，爆发极高但脆皮",
		HPBonus: 0, IntBonus: 40, AgiBonus: 10, LuckBonus: 5, PDefBonus: 5, MDefBonus: 20, StrBonus: 5,
		SkillPool: []string{"火球术", "冰冻术", "雷电术", "元素风暴", "奥术飞弹", "时间停滞", "魔法镜像"},
	},
	{
		Name: ClassAssassin, Emoji: "🗡️", Description: "夜影中的死神，暴击伤害骇人，一击必杀",
		HPBonus: 10, AgiBonus: 40, StrBonus: 20, LuckBonus: 20, PDefBonus: 5, MDefBonus: 5, IntBonus: 5,
		SkillPool: []string{"背刺", "毒刃", "致命打击", "烟雾弹", "连环刺", "暗影步", "诡异一击"},
	},
	{
		Name: ClassPriest, Emoji: "✨", Description: "神明的使者，治疗量惊人，打持久战的专家",
		HPBonus: 30, IntBonus: 25, LuckBonus: 15, MDefBonus: 20, StrBonus: 5, AgiBonus: 10, PDefBonus: 10,
		SkillPool: []string{"神圣之光", "生命绽放", "生命汲取", "神罚天降", "祝圣护盾", "复活祈祷", "圣光爆"},
	},
	{
		Name: ClassPaladin, Emoji: "🛡️", Description: "正义的化身，攻守均衡，携带神圣技能",
		HPBonus: 50, StrBonus: 15, IntBonus: 15, PDefBonus: 15, MDefBonus: 15, AgiBonus: 10, LuckBonus: 10,
		SkillPool: []string{"神圣打击", "圣甲壁垒", "魔法护盾", "圣光庇护", "圣判", "圣盾反弹", "骑士誓言"},
	},
	{
		Name: ClassRanger, Emoji: "🍀", Description: "命运的宠儿，极高运气，战局诡变难以预测",
		HPBonus: 20, LuckBonus: 45, AgiBonus: 25, StrBonus: 10, IntBonus: 10, PDefBonus: 5, MDefBonus: 5,
		SkillPool: []string{"命运之轮", "天降鸿运", "幸运一击", "幸运祝福", "逆转命运", "赌命一搏", "奇迹时刻"},
	},
}

// GetClassDef 根据职业类型获取职业定义
func GetClassDef(c ClassType) *ClassDef {
	for i := range AllClasses {
		if AllClasses[i].Name == c {
			return &AllClasses[i]
		}
	}
	return &AllClasses[0]
}

// ============================================================
//  Buff/Debuff 状态效果
// ============================================================

type StatusEffect struct {
	Name     string
	Duration int    // 剩余回合数
	Value    int    // 效果强度
	Type     string // dot/hot/stun/shield/atk_up/def_up/atk_down/def_down/dodge_up
}

// ============================================================
//  战斗属性
// ============================================================

type BattleAttr struct {
	OwnerID  int
	Name     string    // 显示名（含职业）
	Class    ClassType // 职业
	HP       int       // 当前血量
	MaxHP    int       // 最大血量
	Str      int
	Int      int
	Agi      int
	Luck     int
	PDef     int
	MDef     int
	Skills   []BattleSkill
	Statuses []StatusEffect // 当前状态
	Combo    int            // 连击计数（连续命中累积）
}

// HasStatus 检查是否有某状态
func (b *BattleAttr) HasStatus(name string) bool {
	for _, s := range b.Statuses {
		if s.Name == name {
			return true
		}
	}
	return false
}

// AddStatus 添加/刷新状态
func (b *BattleAttr) AddStatus(s StatusEffect) {
	for i, existing := range b.Statuses {
		if existing.Name == s.Name {
			b.Statuses[i] = s // 刷新
			return
		}
	}
	b.Statuses = append(b.Statuses, s)
}

// TickStatuses 回合结束时处理状态，返回状态描述
func (b *BattleAttr) TickStatuses() string {
	var sb strings.Builder
	alive := b.Statuses[:0]
	for _, s := range b.Statuses {
		switch s.Type {
		case "dot": // 持续伤害
			b.HP -= s.Value
			if b.HP < 0 {
				b.HP = 0
			}
			sb.WriteString(fmt.Sprintf("  ☠️ 【%s】受到持续伤害 %d！\n", b.Name, s.Value))
		case "hot": // 持续治疗
			heal := s.Value
			if b.HP+heal > b.MaxHP {
				heal = b.MaxHP - b.HP
			}
			b.HP += heal
			sb.WriteString(fmt.Sprintf("  💚 【%s】受到持续治疗 %d！\n", b.Name, heal))
		}
		s.Duration--
		if s.Duration > 0 {
			alive = append(alive, s)
		} else {
			// 状态到期，恢复部分属性
			switch s.Type {
			case "atk_down":
				b.Str += s.Value
				b.Int += s.Value
				sb.WriteString(fmt.Sprintf("  🔓 【%s】攻击削弱状态解除\n", b.Name))
			case "def_down":
				b.PDef += s.Value
				b.MDef += s.Value
				sb.WriteString(fmt.Sprintf("  🔓 【%s】防御削弱状态解除\n", b.Name))
			}
		}
	}
	b.Statuses = alive
	return sb.String()
}

// ============================================================
//  技能系统（全新版）
// ============================================================

// SkillEffectFn 技能效果函数签名
// 返回：伤害值（负数=治疗，0=无伤害），日志描述
type SkillEffectFn func(attacker, defender *BattleAttr, round int) (int, string)

type BattleSkill struct {
	Name        string
	Emoji       string
	Description string
	Type        string        // physical/magic/heal/defense/luck/debuff
	Effect      SkillEffectFn `json:"-"`
}

// 生成所有技能池
func buildSkillPool() map[string]BattleSkill {
	pool := map[string]BattleSkill{
		// --------- 战士技能（高力量+高物防，越打越强，主打持久与爆发）---------
		"重击": {
			Name: "重击", Emoji: "💢", Type: "physical",
			Description: "强力单次打击，力量×1.8，必定命中，无视50%物防",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				dmg := int(float64(a.Str)*1.8) - d.PDef/6 // 无视50%物防（除6而非除3）
				if dmg < 20 {
					dmg = 20
				}
				// 取消160上限，战士高力量要能体现出来
				return dmg, fmt.Sprintf("💢 【重击】必中！无视50%%物防，造成 %d 物理伤害", dmg)
			},
		},
		"旋风斩": {
			Name: "旋风斩", Emoji: "🌪️", Type: "physical",
			Description: "横扫攻击，忽视闪避，力量×1.6",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 旋风斩提升到×1.6体现战士爆发，忽视物防/4
				dmg := int(float64(a.Str)*1.6) - d.PDef/4
				if dmg < 15 {
					dmg = 15
				}
				return dmg, fmt.Sprintf("🌪️ 【旋风斩】横扫造成 %d 物理伤害（忽视闪避）", dmg)
			},
		},
		"破甲一击": {
			Name: "破甲一击", Emoji: "🪓", Type: "physical",
			Description: "无视全部物防，力量×1.5，并降低对方物防35%",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				dmg := int(float64(a.Str) * 1.5) // 提高到×1.5，彰显战士破甲专精
				reduction := d.PDef * 35 / 100   // 破甲率提升到35%
				d.PDef -= reduction
				if d.PDef < 0 {
					d.PDef = 0
				}
				d.AddStatus(StatusEffect{Name: "破甲", Type: "def_down", Duration: 2, Value: reduction})
				return dmg, fmt.Sprintf("🪓 【破甲一击】无视物防造成 %d 伤害，对方物防降低 %d（持续2回合）", dmg, reduction)
			},
		},
		"铁壁防御": {
			Name: "铁壁防御", Emoji: "🛡️", Type: "defense",
			Description: "战士专属：物防+60并持续2回合，同时反弹本回合受到的20%伤害",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 战士物防本身就高，提升更多才有意义；同时加反弹盾让防守有攻击性
				a.PDef += 60
				a.AddStatus(StatusEffect{Name: "铁壁", Type: "def_up", Duration: 2, Value: 60})
				a.AddStatus(StatusEffect{Name: "反弹盾", Type: "shield", Duration: 1, Value: 20})
				return 0, "🛡️ 【铁壁防御】铸就铁壁！物防+60（2回合），下次受击反弹20%伤害"
			},
		},
		"战嚎激励": {
			Name: "战嚎激励", Emoji: "📣", Type: "physical",
			Description: "战嚎激发斗志，力量+35，持续3回合，并造成小伤害威慑对手",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 战士天赋加成提升：持续3回合（比原来2回合更久），力量加成提升到35
				a.Str += 35
				a.AddStatus(StatusEffect{Name: "战嚎", Type: "atk_up", Duration: 3, Value: 35})
				dmg := a.Str / 4
				return dmg, fmt.Sprintf("📣 【战嚎激励】发出战嚎！力量+35（3回合），顺势威压造成 %d 伤害", dmg)
			},
		},
		"无敌霸体": {
			Name: "无敌霸体", Emoji: "💎", Type: "defense",
			Description: "战士绝技：下一次受击完全免伤，并反弹40%伤害给对方",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 反弹率从30%提升到40%，战士的招牌大招要有存在感
				a.AddStatus(StatusEffect{Name: "无敌", Type: "shield", Duration: 1, Value: 40})
				return 0, "💎 【无敌霸体】进入无敌状态！下次受击完全免伤并反弹40%伤害！"
			},
		},
		"复仇之刃": {
			Name: "复仇之刃", Emoji: "🔥", Type: "physical",
			Description: "战士越战越勇，血量越低伤害越高，最低血量时力量×3.5",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				ratio := 1.0 - float64(a.HP)/float64(a.MaxHP)
				multi := 1.0 + ratio*2.5 // 最高倍率提升到3.5（血量=0时）
				if multi > 3.5 {
					multi = 3.5
				}
				dmg := int(float64(a.Str) * multi)
				return dmg, fmt.Sprintf("🔥 【复仇之刃】越战越勇！当前%.0f%%血量，造成 %d 物理伤害(%.1f倍)", (1-ratio)*100, dmg, multi)
			},
		},

		// --------- 法师技能（高智力玻璃炮，爆发极高但无防御手段，依赖控制）---------
		"火球术": {
			Name: "火球术", Emoji: "🔥", Type: "magic",
			Description: "爆炎火球，智力×1.8，灼烧3回合，法师高智力核心输出",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 法师智力60+，×1.8才能体现玻璃炮爆发；魔防减弱到1/5
				dmg := int(float64(a.Int)*1.8) - d.MDef/5
				if dmg < 20 {
					dmg = 20
				}
				burnDmg := a.Int / 6 // 灼烧伤害基于智力1/6，法师高智力灼烧也很痛
				if burnDmg < 8 {
					burnDmg = 8
				}
				d.AddStatus(StatusEffect{Name: "灼烧", Type: "dot", Duration: 3, Value: burnDmg}) // 灼烧延长到3回合
				return dmg, fmt.Sprintf("🔥 【火球术】爆炎命中！造成 %d 魔法伤害，灼烧3回合（每回合-%d）", dmg, burnDmg)
			},
		},
		"冰冻术": {
			Name: "冰冻术", Emoji: "❄️", Type: "magic",
			Description: "冰霜爆裂，智力×1.4+减速，目标冻结1回合且魔防下降20%",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 法师的控制技能：冰冻+破魔防，两种效果配合法师连续输出
				dmg := int(float64(a.Int)*1.4) - d.MDef/5
				if dmg < 15 {
					dmg = 15
				}
				mdefDown := d.MDef * 20 / 100
				d.MDef -= mdefDown
				if d.MDef < 0 {
					d.MDef = 0
				}
				d.AddStatus(StatusEffect{Name: "冻结", Type: "stun", Duration: 1, Value: 0})
				d.AddStatus(StatusEffect{Name: "寒冰", Type: "def_down", Duration: 2, Value: mdefDown})
				return dmg, fmt.Sprintf("❄️ 【冰冻术】造成 %d 伤害，目标冻结1回合，魔防下降 %d（2回合）", dmg, mdefDown)
			},
		},
		"雷电术": {
			Name: "雷电术", Emoji: "⚡", Type: "magic",
			Description: "天雷斩击，智力×1.5，固定暴击率+40%，暴击伤害×2.2",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 法师专属高暴击魔法，暴击倍率提高到2.2
				base := int(float64(a.Int)*1.5) - d.MDef/5
				if base < 15 {
					base = 15
				}
				critChance := 40 // 固定40%，法师自带高暴击（不依赖运气，运气是游侠专属）
				if rand.Intn(100) < critChance {
					critDmg := int(float64(base) * (2.2 + rand.Float64()*0.3))
					return critDmg, fmt.Sprintf("⚡⚡ 【雷电术】雷霆暴击！造成 %d 魔法伤害（暴击2.2x）", critDmg)
				}
				return base, fmt.Sprintf("⚡ 【雷电术】天雷斩击，造成 %d 魔法伤害", base)
			},
		},
		"元素风暴": {
			Name: "元素风暴", Emoji: "🌊", Type: "magic",
			Description: "法师终极：4~5轮元素爆发，每次智力×0.8，总伤害极高",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 法师大招：提高每轮倍率到0.8，命中4-5次，总伤害远超其他技能
				hits := 4 + rand.Intn(2) // 4-5次
				total := 0
				parts := make([]string, hits)
				for i := 0; i < hits; i++ {
					dmg := int(float64(a.Int)*0.8) + rand.Intn(15)
					total += dmg
					parts[i] = fmt.Sprintf("%d", dmg)
				}
				return total, fmt.Sprintf("🌊 【元素风暴】%d次元素爆发！%s = 总计 %d 伤害", hits, strings.Join(parts, "+"), total)
			},
		},
		"奥术飞弹": {
			Name: "奥术飞弹", Emoji: "✨", Type: "magic",
			Description: "5枚奥术飞弹，完全无视魔防，每枚智力×0.55",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 飞弹数量增加到5枚，每枚系数提升，强化无视魔防的穿透感
				dmg := int(float64(a.Int)*0.55) * 5
				return dmg, fmt.Sprintf("✨ 【奥术飞弹】5枚飞弹穿透！完全无视魔防，造成 %d 纯粹伤害", dmg)
			},
		},
		"时间停滞": {
			Name: "时间停滞", Emoji: "⏳", Type: "magic",
			Description: "凝固时间，目标跳过下2回合行动，同时造成智力×1.0伤害",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 时间停滞是法师最强控制，停滞时间延长到2回合
				dmg := int(float64(a.Int) * 1.0)
				d.AddStatus(StatusEffect{Name: "停滞", Type: "stun", Duration: 2, Value: 0})
				return dmg, fmt.Sprintf("⏳ 【时间停滞】时间凝固！造成 %d 伤害，对方跳过接下来2回合！", dmg)
			},
		},
		"魔法镜像": {
			Name: "魔法镜像", Emoji: "🪞", Type: "defense",
			Description: "法师唯一防御：制造幻像，反弹下一次攻击100%伤害，并同时回复智力×0.5血量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 法师没有物防，镜像是唯一自保手段，增加一点回血让它更有价值
				a.AddStatus(StatusEffect{Name: "镜像", Type: "shield", Duration: 1, Value: 100})
				heal := a.Int / 2
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return 0, fmt.Sprintf("🪞 【魔法镜像】幻像成形！将反弹100%%伤害，同时回复 %d 生命", heal)
			},
		},

		// --------- 刺客技能（高敏捷+高暴击，技能全靠 Agi 和 Luck，爆发秒杀型）---------
		"背刺": {
			Name: "背刺", Emoji: "🗡️", Type: "physical",
			Description: "偷袭致命弱点，暴击率=Luck+60%，暴击伤害×3.0，未暴击也有Str+Agi×0.6",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 刺客高敏捷，背刺加入敏捷加成；暴击倍率从2.5提升到3.0体现一击必杀
				critChance := a.Luck + 60
				base := int(float64(a.Str)*1.0 + float64(a.Agi)*0.6)
				if rand.Intn(100) < critChance {
					dmg := int(float64(base) * 3.0)
					return dmg, fmt.Sprintf("🗡️💥 【背刺】致命偷袭暴击！(Str+Agi混合)造成 %d 伤害", dmg)
				}
				return base, fmt.Sprintf("🗡️ 【背刺】偷袭命中！造成 %d 伤害（Str+Agi混合）", base)
			},
		},
		"毒刃": {
			Name: "毒刃", Emoji: "☠️", Type: "debuff",
			Description: "淬毒刀刃，造成伤害并附加3回合剧毒，毒伤基于敏捷而非力量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 刺客的毒应基于敏捷（而非力量），体现刺客轻巧上毒的特性
				dmg := int(float64(a.Str) * 0.9)
				poisonDmg := a.Agi/5 + 10 // 毒伤基于敏捷
				d.AddStatus(StatusEffect{Name: "中毒", Type: "dot", Duration: 3, Value: poisonDmg})
				return dmg, fmt.Sprintf("☠️ 【毒刃】淬毒命中！造成 %d 伤害并注入剧毒（每回合-%d，持续3回合）", dmg, poisonDmg)
			},
		},
		"致命打击": {
			Name: "致命打击", Emoji: "💀", Type: "physical",
			Description: "刺客必杀技，命中率=(Luck/3+35)%，命中造成Str×3.5极限伤害",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 高运气刺客命中率更高，命中伤害从×3.0提升到×3.5
				hitRate := 35 + a.Luck/3
				if hitRate > 70 {
					hitRate = 70 // 上限70%，保持赌博感
				}
				if rand.Intn(100) < hitRate {
					dmg := int(float64(a.Str) * 3.5)
					if dmg > 300 {
						dmg = 300
					}
					return dmg, fmt.Sprintf("💀💥 【致命打击】穿透心脏！造成毁灭性的 %d 伤害！！（命中率%d%%）", dmg, hitRate)
				}
				return 0, fmt.Sprintf("💀 【致命打击】出手！但被对方灵巧躲过...（命中率%d%%）", hitRate)
			},
		},
		"烟雾弹": {
			Name: "烟雾弹", Emoji: "💨", Type: "defense",
			Description: "烟雾掩护+剧毒，闪避大幅提升（+Agi×0.8），同时给对方施放微量毒雾",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 修复：原来+80是固定值，改为基于刺客自身敏捷，高敏刺客闪避更强
				dodgeBonus := int(float64(a.Agi) * 0.8)
				if dodgeBonus < 40 {
					dodgeBonus = 40
				}
				a.Agi += dodgeBonus
				a.AddStatus(StatusEffect{Name: "烟雾", Type: "dodge_up", Duration: 2, Value: dodgeBonus / 2})
				// 附加对方小毒
				microPoison := 5
				d.AddStatus(StatusEffect{Name: "毒雾", Type: "dot", Duration: 2, Value: microPoison})
				return 0, fmt.Sprintf("💨 【烟雾弹】毒雾弥漫！敏捷+%d（本回合），对方陷入毒雾（每回合-%d，2回合）", dodgeBonus, microPoison)
			},
		},
		"连环刺": {
			Name: "连环刺", Emoji: "⚡", Type: "physical",
			Description: "6次连刺，每次 Str×0.45+Agi×0.15，连击数越多每刺越痛",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 刺客专属：连刺加入敏捷加成，高敏刺客连刺总伤极高
				total := 0
				var parts []string
				for i := 0; i < 6; i++ {
					dmg := int(float64(a.Str)*0.45+float64(a.Agi)*0.15) + a.Combo*6
					total += dmg
					parts = append(parts, fmt.Sprintf("%d", dmg))
					a.Combo++
				}
				return total, fmt.Sprintf("⚡ 【连环刺】六连刺！%s = 总计 %d 伤害（连击×%d）", strings.Join(parts, "+"), total, a.Combo)
			},
		},
		"暗影步": {
			Name: "暗影步", Emoji: "🌑", Type: "physical",
			Description: "影分身突袭，(Str+Agi)×1.0无视全防御，且本回合完全无法被闪避",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 提高系数到×1.0，且明确说明无法被闪避
				dmg := int(float64(a.Str+a.Agi) * 1.0)
				return dmg, fmt.Sprintf("🌑 【暗影步】穿越阴影！无视防御与闪避，造成 %d 纯粹伤害（Str+Agi）", dmg)
			},
		},
		"诡异一击": {
			Name: "诡异一击", Emoji: "🎭", Type: "debuff",
			Description: "诅咒攻击，降低对方力量30%+智力30%+敏捷20%（2回合），造成Agi×0.8伤害",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 增加敏捷削弱，刺客诡异一击专门针对敏捷高的对手；伤害改为敏捷为主
				strDown := d.Str * 30 / 100
				intDown := d.Int * 30 / 100
				agiDown := d.Agi * 20 / 100
				d.Str -= strDown
				d.Int -= intDown
				d.Agi -= agiDown
				if d.Agi < 0 {
					d.Agi = 0
				}
				d.AddStatus(StatusEffect{Name: "诅咒虚弱", Type: "atk_down", Duration: 2, Value: (strDown + intDown) / 2})
				dmg := int(float64(a.Agi) * 0.8)
				return dmg, fmt.Sprintf("🎭 【诡异一击】诅咒命中！力量-%d 智力-%d 敏捷-%d（2回合），造成 %d 伤害", strDown, intDown, agiDown, dmg)
			},
		},

		// --------- 牧师技能（高智力+高魔防+回复，持久战专家，每个技能都有治疗成分）---------
		"神圣之光": {
			Name: "神圣之光", Emoji: "✨", Type: "heal",
			Description: "圣光治愈，回复最大血量的40%+智力×0.3，附带1回合神圣护盾",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 牧师的核心治疗：回复量增加，与智力挂钩
				heal := int(float64(a.MaxHP)*0.40) + int(float64(a.Int)*0.3)
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				a.AddStatus(StatusEffect{Name: "护盾", Type: "shield", Duration: 1, Value: 50})
				return -heal, fmt.Sprintf("✨ 【神圣之光】圣光治愈！回复 %d 生命（含智力加成），附加50点护盾", heal)
			},
		},
		"生命绽放": {
			Name: "生命绽放", Emoji: "🌺", Type: "heal",
			Description: "绽放治疗，回复智力×1.5生命，附加3回合持续治疗（每回合智力×0.2）",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 持续治疗由2回合延长到3回合，HoT量也跟智力挂钩
				heal := int(float64(a.Int) * 1.5)
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				hotVal := a.Int / 5
				if hotVal < 8 {
					hotVal = 8
				}
				a.AddStatus(StatusEffect{Name: "生命之花", Type: "hot", Duration: 3, Value: hotVal})
				return -heal, fmt.Sprintf("🌺 【生命绽放】花瓣绽放！回复 %d 生命，持续治疗3回合（每回合+%d）", heal, hotVal)
			},
		},
		"生命汲取": {
			Name: "生命汲取", Emoji: "🩸", Type: "heal",
			Description: "汲取生命精华，造成智力×1.2伤害，回复70%伤害量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 回血比例从60%提升到70%，更突出牧师的持续性
				dmg := int(float64(a.Int) * 1.2)
				heal := dmg * 70 / 100
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return dmg, fmt.Sprintf("🩸 【生命汲取】精华汲取！造成 %d 伤害，同时回复 %d 生命（70%%转化）", dmg, heal)
			},
		},
		"神罚天降": {
			Name: "神罚天降", Emoji: "⚡", Type: "magic",
			Description: "圣光审判，55%概率：智力×2.5大爆发；45%概率：回复智力×0.8血量（牧师高智力让两个效果都强力）",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 牧师智力高，提升爆发倍率到2.5和回复量到0.8
				if rand.Intn(100) < 55 {
					dmg := int(float64(a.Int) * 2.5)
					return dmg, fmt.Sprintf("⚡✨ 【神罚天降】神明降罚！造成 %d 圣属性伤害！", dmg)
				}
				heal := int(float64(a.Int) * 0.8)
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return -heal, fmt.Sprintf("✨ 【神罚天降】神明庇护！回复 %d 生命（圣光祝福）", heal)
			},
		},
		"祝圣护盾": {
			Name: "祝圣护盾", Emoji: "🔵", Type: "defense",
			Description: "神圣护盾，吸收150点伤害（持续3回合），同时回复智力×0.5血量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 护盾强度从100提升到150，持续时间延长到3回合，回血与智力挂钩
				a.AddStatus(StatusEffect{Name: "护盾", Type: "shield", Duration: 3, Value: 150})
				heal := a.Int / 2
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return -heal, fmt.Sprintf("🔵 【祝圣护盾】神圣守护！吸收150点伤害（3回合），回复 %d 生命", heal)
			},
		},
		"复活祈祷": {
			Name: "复活祈祷", Emoji: "💫", Type: "heal",
			Description: "牧师秘技：血量低于25%时触发奇迹，回复65%最大血量+移除所有负面状态",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 触发阈值提升到25%（更好触发），回复量提升到65%，关键加入移除debuff
				if float64(a.HP) < float64(a.MaxHP)*0.25 {
					heal := int(float64(a.MaxHP) * 0.65)
					if a.HP+heal > a.MaxHP {
						heal = a.MaxHP - a.HP
					}
					a.HP += heal
					// 移除所有负面状态（dot/stun/def_down/atk_down）
					var cleanStatuses []StatusEffect
					for _, s := range a.Statuses {
						if s.Type == "dot" || s.Type == "stun" || s.Type == "def_down" || s.Type == "atk_down" {
							continue
						}
						cleanStatuses = append(cleanStatuses, s)
					}
					a.Statuses = cleanStatuses
					return -heal, fmt.Sprintf("💫 【复活祈祷】奇迹！在生死边缘回复 %d 生命，并清除所有负面状态！", heal)
				}
				// 血量充足时：普通治疗+顺带净化一个debuff
				heal := int(float64(a.MaxHP) * 0.20)
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return -heal, fmt.Sprintf("💫 【复活祈祷】回复 %d 生命（血量充足，效果减弱）", heal)
			},
		},
		"圣光爆": {
			Name: "圣光爆", Emoji: "☀️", Type: "magic",
			Description: "牧师爆发：牺牲35点血量，造成智力×2.0圣属性伤害，命中后回复自身智力×0.4血量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 增加命中后回血，体现牧师攻击同时也在治疗自己的特色
				sacrifice := 35
				a.HP -= sacrifice
				if a.HP < 1 {
					a.HP = 1
				}
				dmg := int(float64(a.Int) * 2.0)
				// 施法后回血
				healBack := a.Int * 4 / 10
				if a.HP+healBack > a.MaxHP {
					healBack = a.MaxHP - a.HP
				}
				a.HP += healBack
				return dmg, fmt.Sprintf("☀️ 【圣光爆】以血换力！牺牲 %d 血量造成 %d 圣属性伤害，回复 %d 生命", sacrifice, dmg, healBack)
			},
		},

		// --------- 圣骑士技能（力量+智力均衡，攻防皆宜，携带神圣技能）---------
		"神圣打击": {
			Name: "神圣打击", Emoji: "✝️", Type: "physical",
			Description: "圣力降世，力量×0.9+智力×0.9，同时弱化对方，穿透物防+魔防各1/7",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 圣骑士均衡型，双属性加成，穿透双防体现神圣属性
				dmg := int(float64(a.Str)*0.9) + int(float64(a.Int)*0.9) - (d.PDef+d.MDef)/7
				if dmg < 20 {
					dmg = 20
				}
				return dmg, fmt.Sprintf("✝️ 【神圣打击】圣力降临！造成 %d 神圣伤害（力量+智力混合，穿透双防）", dmg)
			},
		},
	"圣甲壁垒": {
		Name: "圣甲壁垒", Emoji: "🛡️", Type: "defense",
		Description: "圣骑士专属全防：物防+魔防各+40，持续2回合（圣骑士均衡防御，区别于战士纯物防）",
		Effect: func(a, d *BattleAttr, round int) (int, string) {
			// 圣骑士的铁壁是全防提升，与战士的纯物防不同
			a.PDef += 40
			a.MDef += 40
			a.AddStatus(StatusEffect{Name: "圣甲", Type: "def_up", Duration: 2, Value: 40})
			return 0, "🛡️ 【圣甲壁垒】圣盾加持！物防+40 魔防+40（2回合），全面防御提升"
		},
	},
		"魔法护盾": {
			Name: "魔法护盾", Emoji: "🔮", Type: "defense",
			Description: "圣魔护盾，魔防+50持续2回合，并反射10%受到的魔法伤害",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 魔防提升到+50，加入魔法反射体现圣骑士的圣属性
				a.MDef += 50
				a.AddStatus(StatusEffect{Name: "魔盾", Type: "def_up", Duration: 2, Value: 50})
				a.AddStatus(StatusEffect{Name: "反弹盾", Type: "shield", Duration: 2, Value: 10})
				return 0, "🔮 【魔法护盾】圣魔护盾！魔防+50（2回合），同时反弹受到魔法伤害的10%"
			},
		},
	"圣光庇护": {
		Name: "圣光庇护", Emoji: "✨", Type: "heal",
		Description: "圣骑士治愈：回复最大血量30%+力量×0.5（与牧师的智力加成版区分，体现圣骑士力量特色）",
		Effect: func(a, d *BattleAttr, round int) (int, string) {
			// 圣骑士版：回复量和力量挂钩（而非像牧师那样和智力挂钩）
			heal := int(float64(a.MaxHP)*0.30) + int(float64(a.Str)*0.5)
			if a.HP+heal > a.MaxHP {
				heal = a.MaxHP - a.HP
			}
			a.HP += heal
			return -heal, fmt.Sprintf("✨ 【圣光庇护】圣光垂怜！回复 %d 生命（含力量加成）", heal)
		},
	},
		"圣判": {
			Name: "圣判", Emoji: "⚖️", Type: "physical",
			Description: "正义审判，对HP高于自己的目标造成(Str+Int)×1.2，额外×1.6倍惩戒加成",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 提高基础伤害到×1.2，惩戒加成倍率从1.5提升到1.6
				base := int(float64(a.Str+a.Int) * 1.2)
				if d.HP > a.HP {
					base = int(float64(base) * 1.6)
					return base, fmt.Sprintf("⚖️ 【圣判】审判血量更高的敌人！造成 %d 神圣伤害（惩戒×1.6）", base)
				}
				return base, fmt.Sprintf("⚖️ 【圣判】正义审判！造成 %d 神圣伤害", base)
			},
		},
		"圣盾反弹": {
			Name: "圣盾反弹", Emoji: "🔁", Type: "defense",
			Description: "举起圣盾，1回合完全反弹受击伤害的60%（圣骑士专精比战士更强的反弹）",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 圣骑士的圣盾反弹率从50%提升到60%，体现均衡型防守的优势
				a.AddStatus(StatusEffect{Name: "反弹盾", Type: "shield", Duration: 1, Value: 60})
				return 0, "🔁 【圣盾反弹】举起神圣之盾！反弹下次受击伤害的60%！（远超普通反弹盾）"
			},
		},
		"骑士誓言": {
			Name: "骑士誓言", Emoji: "🏅", Type: "heal",
			Description: "圣骑士终极：誓言强化，回复30%血量，力量+20 智力+20 物防+15 魔防+15（永久强化）",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 更全面的永久强化，体现圣骑士越战越强的后期属性
				heal := int(float64(a.MaxHP) * 0.30)
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				a.Str += 20
				a.Int += 20
				a.PDef += 15
				a.MDef += 15
				return -heal, fmt.Sprintf("🏅 【骑士誓言】誓言强化！回复 %d 生命，力量+20 智力+20 物防+15 魔防+15（永久）", heal)
			},
		},

		// --------- 游侠技能（极高运气+中等敏捷，所有技能都基于Luck，命运诡变难以预测）---------
		"命运之轮": {
			Name: "命运之轮", Emoji: "🎡", Type: "luck",
			Description: "旋转命运之轮，伤害完全由运气决定：Luck×0.5 ~ Luck×3.5",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 游侠高运气，范围更广更戏剧性：最低0.5x最高3.5x
				minDmg := a.Luck / 2
				maxDmg := a.Luck * 35 / 10
				if minDmg < 20 {
					minDmg = 20
				}
				if maxDmg < minDmg+30 {
					maxDmg = minDmg + 30
				}
				dmg := minDmg + rand.Intn(maxDmg-minDmg+1)
				return dmg, fmt.Sprintf("🎡 【命运之轮】命运转动！Luck=%d，范围[%d~%d]，造成 %d 伤害", a.Luck, minDmg, maxDmg, dmg)
			},
		},
		"天降鸿运": {
			Name: "天降鸿运", Emoji: "🎉", Type: "luck",
			Description: "65%概率：运气×3.0大伤害；35%概率：回复运气×2.0血量",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 提高爆发倍率到3.0，运气高的游侠输出极为恐怖
				if rand.Intn(100) < 65 {
					dmg := int(float64(a.Luck) * 3.0)
					return dmg, fmt.Sprintf("🎉 【天降鸿运】鸿运当头！Luck=%d，造成 %d 伤害（×3.0）", a.Luck, dmg)
				}
				heal := a.Luck * 2
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return -heal, fmt.Sprintf("🎉 【天降鸿运】神明庇佑！回复 %d 生命（Luck×2.0）", heal)
			},
		},
		"幸运一击": {
			Name: "幸运一击", Emoji: "🍀", Type: "luck",
			Description: "运气×2.0伤害，暴击率=Luck%，暴击则额外触发一次运气×1.5追加打击",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				base := a.Luck * 2
				if rand.Intn(100) < a.Luck {
					extra := int(float64(a.Luck) * 1.5)
					total := base + extra
					return total, fmt.Sprintf("🍀🍀 【幸运一击】超级幸运！%d + 追加 %d = 总计 %d 伤害", base, extra, total)
				}
				return base, fmt.Sprintf("🍀 【幸运一击】造成 %d 伤害（Luck×2.0）", base)
			},
		},
		"幸运祝福": {
			Name: "幸运祝福", Emoji: "🌈", Type: "luck",
			Description: "运气爆发，运气+80，持续3回合，同时回复运气×0.4血量（高运气游侠极强）",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 运气加成从60提升到80，持续3回合，回血量也与运气挂钩
				a.Luck += 80
				a.AddStatus(StatusEffect{Name: "幸运祝福", Type: "atk_up", Duration: 3, Value: 80})
				heal := a.Luck * 4 / 10
				if a.HP+heal > a.MaxHP {
					heal = a.MaxHP - a.HP
				}
				a.HP += heal
				return -heal, fmt.Sprintf("🌈 【幸运祝福】运气飙升！运气+80（3回合），回复 %d 生命（Luck×0.4）", heal)
			},
		},
		"逆转命运": {
			Name: "逆转命运", Emoji: "♻️", Type: "luck",
			Description: "反转之力，血量<30%时：Luck×4.0+Str×1.0 超强爆发；正常时：Luck×2.0",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 更彰显游侠"越绝望越强"，低血量时加入力量加成，最终伤害极高
				if float64(a.HP) < float64(a.MaxHP)*0.3 {
					dmg := int(float64(a.Luck)*4.0) + int(float64(a.Str)*1.0)
					return dmg, fmt.Sprintf("♻️ 【逆转命运】置之死地而后生！Luck×4+Str×1 = %d 伤害（绝境爆发）", dmg)
				}
				dmg := a.Luck * 2
				return dmg, fmt.Sprintf("♻️ 【逆转命运】命运反转！造成 %d 伤害（Luck×2.0）", dmg)
			},
		},
		"赌命一搏": {
			Name: "赌命一搏", Emoji: "🎲", Type: "luck",
			Description: "押注命运！1/6爆出600伤害；运气高则有额外救援：>70运气时自救+100血",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 爆发伤害从500提升到600，高运气游侠有自救补偿机制
				roll := rand.Intn(6) + 1
				if roll == 6 {
					return 600, "🎲🎰 【赌命一搏】投出了6！！命运眷顾，爆发 600 点极限伤害！！！"
				}
				selfDmg := 80
				a.HP -= selfDmg
				if a.HP < 0 {
					a.HP = 0
				}
				// 高运气有救援
				if a.Luck > 70 {
					rescue := 100
					if a.HP+rescue > a.MaxHP {
						rescue = a.MaxHP - a.HP
					}
					a.HP += rescue
					return 0, fmt.Sprintf("🎲 【赌命一搏】投出了 %d... 赌输了，受到 %d 伤害，但运气保佑，回复 %d 生命", roll, selfDmg, rescue)
				}
				return 0, fmt.Sprintf("🎲 【赌命一搏】投出了 %d... 赌输了，自身受到 %d 伤害", roll, selfDmg)
			},
		},
		"奇迹时刻": {
			Name: "奇迹时刻", Emoji: "🌟", Type: "luck",
			Description: "运气≥70触发奇迹：回复55%血量+造成Luck×3.0伤害；运气不足则3选1随机效果",
			Effect: func(a, d *BattleAttr, round int) (int, string) {
				// 触发门槛调低到70（游侠的Luck加成后很容易达到），奇迹伤害改为Luck×3
				if a.Luck >= 70 {
					heal := int(float64(a.MaxHP) * 0.55)
					if a.HP+heal > a.MaxHP {
						heal = a.MaxHP - a.HP
					}
					a.HP += heal
					dmg := int(float64(a.Luck) * 3.0)
					return dmg, fmt.Sprintf("🌟✨ 【奇迹时刻】奇迹降临（Luck=%d）！回复 %d 生命 + 造成 %d 伤害！！", a.Luck, heal, dmg)
				}
				r := rand.Intn(3)
				switch r {
				case 0:
					dmg := a.Luck + rand.Intn(80)
					return dmg, fmt.Sprintf("🌟 【奇迹时刻】随机触发：造成 %d 伤害", dmg)
				case 1:
					heal := a.Luck / 2
					if a.HP+heal > a.MaxHP {
						heal = a.MaxHP - a.HP
					}
					a.HP += heal
					return -heal, fmt.Sprintf("🌟 【奇迹时刻】随机触发：回复 %d 生命（Luck÷2）", heal)
				default:
					a.Luck += 40
					return 0, fmt.Sprintf("🌟 【奇迹时刻】随机触发：运气+40（现在Luck=%d）", a.Luck)
				}
			},
		},
	}
	return pool
}

// buildClassSkills 根据职业构建技能列表（3个技能）
func buildClassSkills(cls ClassType) []BattleSkill {
	pool := buildSkillPool()
	def := GetClassDef(cls)
	var skills []BattleSkill
	for _, name := range def.SkillPool {
		if s, ok := pool[name]; ok {
			skills = append(skills, s)
		}
	}
	// 随机选3个
	rand.Shuffle(len(skills), func(i, j int) { skills[i], skills[j] = skills[j], skills[i] })
	if len(skills) > 3 {
		skills = skills[:3]
	}
	return skills
}

// ============================================================
//  属性生成
// ============================================================

// GenerateBattleAttr 生成战斗属性（带职业加成）
func GenerateBattleAttr(userID int, username string, cls ClassType) *BattleAttr {
	def := GetClassDef(cls)

	// 基础随机属性池：每人共有240点随机属性可分配
	totalPoints := 240
	attrNames := []string{"str", "int", "agi", "luck", "pdef", "mdef"}
	attrVals := map[string]*int{}
	str, intV, agi, luck, pdef, mdef := 20, 20, 20, 20, 20, 20
	attrVals["str"] = &str
	attrVals["int"] = &intV
	attrVals["agi"] = &agi
	attrVals["luck"] = &luck
	attrVals["pdef"] = &pdef
	attrVals["mdef"] = &mdef
	remaining := totalPoints
	for remaining > 0 {
		key := attrNames[rand.Intn(len(attrNames))]
		add := 1 + rand.Intn(15)
		if add > remaining {
			add = remaining
		}
		if *attrVals[key]+add > 90 {
			add = 90 - *attrVals[key]
		}
		if add <= 0 {
			continue
		}
		*attrVals[key] += add
		remaining -= add
	}

	// 叠加职业加成
	str += def.StrBonus
	intV += def.IntBonus
	agi += def.AgiBonus
	luck += def.LuckBonus
	pdef += def.PDefBonus
	mdef += def.MDefBonus

	baseHP := 200 + def.HPBonus + rand.Intn(61) // 200~260 + 职业加成

	attr := &BattleAttr{
		OwnerID: userID,
		Name:    fmt.Sprintf("%s·%s%s", username, def.Emoji, string(def.Name)),
		Class:   cls,
		HP:      baseHP,
		MaxHP:   baseHP,
		Str:     str,
		Int:     intV,
		Agi:     agi,
		Luck:    luck,
		PDef:    pdef,
		MDef:    mdef,
		Skills:  buildClassSkills(cls),
	}
	return attr
}

// ============================================================
//  战斗核心
// ============================================================

// init 初始化决斗房间
func init() {
	duels = make(map[string]*Duel)
	duelsByTime = make([]*Duel, 0)
}

// Duel 决斗房间
type Duel struct {
	ID              string
	Initiator       int
	InitiatorAttrs  *BattleAttr
	InitiatorClass  ClassType
	Challenger      int
	ChallengerAttrs *BattleAttr
	ChallengerClass ClassType
	Status          string // waiting/choosing/fighting/ended
	CreateTime      time.Time
	Platform        string
	Bet             int // 下注积分
}

// ============================================================
//  普通攻击
// ============================================================

func normalAttack(attacker, defender *BattleAttr) (int, string) {
	var dmg int
	var log string
	if attacker.Str >= attacker.Int {
		// 物理攻击
		base := int(float64(attacker.Str) * (1.2 - float64(defender.PDef)/200.0))
		if base < 10 {
			base = 10
		}
		dmg = base
		log = fmt.Sprintf("  ⚔️ 普通物理攻击（力量%d vs 物防%d）→ %d 伤害\n", attacker.Str, defender.PDef, dmg)
	} else {
		// 魔法攻击
		base := int(float64(attacker.Int) * (1.1 - float64(defender.MDef)/200.0))
		if base < 10 {
			base = 10
		}
		dmg = base
		log = fmt.Sprintf("  🔮 普通魔法攻击（智力%d vs 魔防%d）→ %d 伤害\n", attacker.Int, defender.MDef, dmg)
	}

	// 闪避检测
	agiAdv := (defender.Agi - attacker.Agi) / 10
	if agiAdv > 0 {
		dodgeRate := agiAdv * 8
		if dodgeRate > 50 {
			dodgeRate = 50
		}
		if rand.Intn(100) < dodgeRate {
			attacker.Combo = 0
			return 0, fmt.Sprintf("  💨 闪避！（敏捷差%d，闪避率%d%%）\n", agiAdv*10, dodgeRate)
		}
	}

	// 暴击
	if rand.Intn(100) < attacker.Luck/3 {
		mult := 1.4 + rand.Float64()*0.4
		dmg = int(float64(dmg) * mult)
		log += fmt.Sprintf("  🍀 普通攻击暴击！(×%.1f)\n", mult)
	}

	attacker.Combo++
	return dmg, log
}

// ============================================================
//  战斗事件（每场战斗随机触发）
// ============================================================

type BattleEvent struct {
	Name   string
	Desc   string
	Effect func(a, b *BattleAttr) string
}

func getRandomEvent() *BattleEvent {
	events := []BattleEvent{
		{
			Name: "神秘祭坛", Desc: "战场中央出现神秘祭坛",
			Effect: func(a, b *BattleAttr) string {
				bonus := 20 + rand.Intn(31)
				a.Str += bonus
				b.Str += bonus
				return fmt.Sprintf("  🏛️ 【神秘祭坛】双方力量均+%d！", bonus)
			},
		},
		{
			Name: "时间扭曲", Desc: "时间短暂扭曲",
			Effect: func(a, b *BattleAttr) string {
				// 双方下回合伤害×1.5
				a.AddStatus(StatusEffect{Name: "时间扭曲", Type: "atk_up", Duration: 1, Value: 20})
				b.AddStatus(StatusEffect{Name: "时间扭曲", Type: "atk_up", Duration: 1, Value: 20})
				return "  ⏰ 【时间扭曲】时间加速！本回合所有人攻击力+20！"
			},
		},
		{
			Name: "毒雾蔓延", Desc: "战场被毒雾笼罩",
			Effect: func(a, b *BattleAttr) string {
				dmg := 10 + rand.Intn(15)
				a.AddStatus(StatusEffect{Name: "战场毒雾", Type: "dot", Duration: 2, Value: dmg})
				b.AddStatus(StatusEffect{Name: "战场毒雾", Type: "dot", Duration: 2, Value: dmg})
				return fmt.Sprintf("  ☠️ 【毒雾蔓延】毒雾笼罩战场！双方每回合受到 %d 持续伤害（2回合）", dmg)
			},
		},
		{
			Name: "圣水洗礼", Desc: "圣水从天而降",
			Effect: func(a, b *BattleAttr) string {
				healA := 40 + rand.Intn(41)
				healB := 40 + rand.Intn(41)
				if a.HP+healA > a.MaxHP {
					healA = a.MaxHP - a.HP
				}
				if b.HP+healB > b.MaxHP {
					healB = b.MaxHP - b.HP
				}
				a.HP += healA
				b.HP += healB
				return fmt.Sprintf("  💧 【圣水洗礼】圣水降临！双方回血 %d / %d", healA, healB)
			},
		},
		{
			Name: "狂暴之怒", Desc: "狂暴气息涌来",
			Effect: func(a, b *BattleAttr) string {
				a.Str += 15
				a.Int += 15
				b.Str += 15
				b.Int += 15
				return "  😡 【狂暴之怒】狂暴气息！双方攻击属性全面+15！"
			},
		},
		{
			Name: "陨石天降", Desc: "一颗陨石随机砸向一人",
			Effect: func(a, b *BattleAttr) string {
				dmg := 60 + rand.Intn(61)
				if rand.Intn(2) == 0 {
					a.HP -= dmg
					if a.HP < 0 {
						a.HP = 0
					}
					return fmt.Sprintf("  ☄️ 【陨石天降】陨石砸中了 %s！受到 %d 伤害！", a.Name, dmg)
				}
				b.HP -= dmg
				if b.HP < 0 {
					b.HP = 0
				}
				return fmt.Sprintf("  ☄️ 【陨石天降】陨石砸中了 %s！受到 %d 伤害！", b.Name, dmg)
			},
		},
	}
	return &events[rand.Intn(len(events))]
}

// ============================================================
//  战斗动画记录
// ============================================================

// BattleAction 记录单次行动
type BattleAction struct {
	Attacker string `json:"attacker"`
	Skill   string `json:"skill"`
	Type    string `json:"type"` // physical/magic/heal/defense
	Damage  int    `json:"damage"`
	Heal    int    `json:"heal"`
	Crit    bool   `json:"crit"`
	Dodge   bool   `json:"dodge"`
}

// BattleRound 记录单回合
type BattleRound struct {
	Number   int           `json:"number"`
	Actions  []BattleAction `json:"actions"`
	Event    string        `json:"event,omitempty"`
}

// BattleRecord 完整战斗记录（供HTML动画使用）
type BattleRecord struct {
	Initiator struct {
		Name      string `json:"name"`
		ClassType string `json:"classType"`
		InitHP    int    `json:"initHp"`   // 战斗开始时的HP（用于动画播放）
		FinalHP   int    `json:"finalHp"` // 战斗结束时的HP
		MaxHp     int    `json:"maxHp"`
	} `json:"initiator"`
	Challenger struct {
		Name      string `json:"name"`
		ClassType string `json:"classType"`
		InitHP    int    `json:"initHp"`   // 战斗开始时的HP（用于动画播放）
		FinalHP   int    `json:"finalHp"` // 战斗结束时的HP
		MaxHp     int    `json:"maxHp"`
	} `json:"challenger"`
	Rounds []BattleRound `json:"rounds"`
	Bet    int           `json:"bet"`
	Winner string        `json:"winner"`
}

// ============================================================
//  主战斗逻辑
// ============================================================

func RunDuelBattle(duelID string, duel *Duel, sender *Sender) {
	duelMutex.Lock()
	defer duelMutex.Unlock()

	// 初始化战斗记录
	var record BattleRecord
	record.Initiator.Name = duel.InitiatorAttrs.Name
	record.Initiator.ClassType = string(duel.InitiatorClass)
	record.Initiator.MaxHp = duel.InitiatorAttrs.MaxHP
	record.Initiator.InitHP = duel.InitiatorAttrs.HP
	record.Initiator.FinalHP = duel.InitiatorAttrs.HP
	record.Challenger.Name = duel.ChallengerAttrs.Name
	record.Challenger.ClassType = string(duel.ChallengerClass)
	record.Challenger.MaxHp = duel.ChallengerAttrs.MaxHP
	record.Challenger.InitHP = duel.ChallengerAttrs.HP
	record.Challenger.FinalHP = duel.ChallengerAttrs.HP
	record.Bet = duel.Bet
	if record.Bet <= 0 {
		record.Bet = 50
	}

	a := *duel.InitiatorAttrs   // 深拷贝
	b := *duel.ChallengerAttrs  // 深拷贝
	// 拷贝技能切片
	aSkills := make([]BattleSkill, len(duel.InitiatorAttrs.Skills))
	bSkills := make([]BattleSkill, len(duel.ChallengerAttrs.Skills))
	copy(aSkills, duel.InitiatorAttrs.Skills)
	copy(bSkills, duel.ChallengerAttrs.Skills)
	a.Skills = aSkills
	b.Skills = bSkills

	var sb strings.Builder

	// ---- 战斗头部 ----
	sb.WriteString("╔══════════════════════╗\n")
	sb.WriteString("║      ⚔️  决斗开始  ⚔️      ║\n")
	sb.WriteString("╚══════════════════════╝\n\n")

	sb.WriteString(fmt.Sprintf("🔴 %s  vs  🔵 %s\n\n", a.Name, b.Name))

	sb.WriteString("━━━━━ 属性面板 ━━━━━\n")
	sb.WriteString(fmt.Sprintf("❤️ 生命：%d  vs  %d\n", a.MaxHP, b.MaxHP))
	sb.WriteString(fmt.Sprintf("💪 力量：%d  vs  %d\n", a.Str, b.Str))
	sb.WriteString(fmt.Sprintf("🧠 智力：%d  vs  %d\n", a.Int, b.Int))
	sb.WriteString(fmt.Sprintf("⚡ 敏捷：%d  vs  %d\n", a.Agi, b.Agi))
	sb.WriteString(fmt.Sprintf("🍀 运气：%d  vs  %d\n", a.Luck, b.Luck))
	sb.WriteString(fmt.Sprintf("🛡️ 物防：%d  vs  %d\n", a.PDef, b.PDef))
	sb.WriteString(fmt.Sprintf("🔮 魔防：%d  vs  %d\n\n", a.MDef, b.MDef))

	sb.WriteString("━━━━━ 技能配置 ━━━━━\n")
	sb.WriteString(fmt.Sprintf("🔴 %s 的技能：", a.Name))
	for _, sk := range a.Skills {
		sb.WriteString(fmt.Sprintf("[%s%s] ", sk.Emoji, sk.Name))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("🔵 %s 的技能：", b.Name))
	for _, sk := range b.Skills {
		sb.WriteString(fmt.Sprintf("[%s%s] ", sk.Emoji, sk.Name))
	}
	sb.WriteString("\n\n")

	// 扣除双方积分
	bet := duel.Bet
	if bet <= 0 {
		bet = 50
	}
	db.Model(&User{}).Where("number = ?", duel.Initiator).Update("coin", gorm.Expr("coin - ?", bet))
	db.Model(&User{}).Where("number = ?", duel.Challenger).Update("coin", gorm.Expr("coin - ?", bet))
	RecordCoinLog(duel.Initiator, -bet, "游戏", "决斗下注", BotContext())
	RecordCoinLog(duel.Challenger, -bet, "游戏", "决斗下注", BotContext())

	// 决定先手（敏捷高者先攻，相等时随机）
	aFirst := a.Agi > b.Agi || (a.Agi == b.Agi && rand.Intn(2) == 0)
	var attacker, defender *BattleAttr
	if aFirst {
		attacker, defender = &a, &b
	} else {
		attacker, defender = &b, &a
	}
	sb.WriteString(fmt.Sprintf("⚡ %s 敏捷更高，获得先手！\n\n", attacker.Name))

	maxRounds := 8
	round := 1
	aSkillCD := make(map[string]int) // 技能CD，key=技能名
	bSkillCD := make(map[string]int)

	// 随机战场事件（每场3~5个，分布在不同回合）
	eventRounds := make(map[int]*BattleEvent)
	evCnt := 2 + rand.Intn(3)
	usedRounds := map[int]bool{}
	for i := 0; i < evCnt; i++ {
		r := 2 + rand.Intn(maxRounds-2)
		if !usedRounds[r] {
			usedRounds[r] = true
			eventRounds[r] = getRandomEvent()
		}
	}

	// 记录本场战斗的所有回合数据
	var allRounds []BattleRound

	for round <= maxRounds && a.HP > 0 && b.HP > 0 {
		var roundActions []BattleAction
		var roundEvent string
		sb.WriteString(fmt.Sprintf("━━━ 第 %d 回合 ━━━\n", round))

		// 战场随机事件
		if ev, ok := eventRounds[round]; ok {
			sb.WriteString(fmt.Sprintf("\n🌟 战场事件：【%s】\n", ev.Name))
			result := ev.Effect(&a, &b)
			sb.WriteString(result + "\n\n")
		}

		// 检查冻结/停滞状态
		if attacker.HasStatus("冻结") || attacker.HasStatus("停滞") {
			sb.WriteString(fmt.Sprintf("  🧊 %s 被冻结/停滞，跳过本回合行动！\n", attacker.Name))
			// 消耗状态
			newStatuses := attacker.Statuses[:0]
			for _, s := range attacker.Statuses {
				if s.Name == "冻结" || s.Name == "停滞" {
					// skip
				} else {
					newStatuses = append(newStatuses, s)
				}
			}
			attacker.Statuses = newStatuses
		} else {
			// 决定行动：技能还是普通攻击
			skillUsed := false
			var skillCD map[string]int
			if attacker == &a {
				skillCD = aSkillCD
			} else {
				skillCD = bSkillCD
			}

			// 更新技能CD
			for k, v := range skillCD {
				if v > 0 {
					skillCD[k] = v - 1
				}
			}

			// 技能使用逻辑（更智能）
			if len(attacker.Skills) > 0 {
				sk := smartSelectSkill(attacker, defender, round, skillCD)
				if sk != nil {
					dmg, log := sk.Effect(attacker, defender, round)
					sb.WriteString(fmt.Sprintf("  🎯 %s 使用技能【%s%s】\n", attacker.Name, sk.Emoji, sk.Name))
					sb.WriteString("  " + log + "\n")
					skillCD[sk.Name] = 2 // 技能有2回合CD

					// 记录动画动作
					action := BattleAction{Attacker: attacker.Name, Skill: sk.Name, Type: sk.Type}
					if strings.Contains(log, "暴击") {
						action.Crit = true
					}
					if strings.Contains(log, "闪避") {
						action.Dodge = true
					}
					if dmg < 0 {
						action.Heal = -dmg
						attacker.HP += -dmg
						if attacker.HP > attacker.MaxHP {
							attacker.HP = attacker.MaxHP
						}
					} else if dmg > 0 {
						dmg = applyDefenseEffects(defender, attacker, dmg, &sb)
						action.Damage = dmg
						if defender == &b {
							b.HP -= dmg
							if b.HP < 0 {
								b.HP = 0
							}
						} else {
							a.HP -= dmg
							if a.HP < 0 {
								a.HP = 0
							}
						}
					}
					roundActions = append(roundActions, action)
					skillUsed = true
				}
			}

			if !skillUsed {
				// 普通攻击
				dmg, log := normalAttack(attacker, defender)
				sb.WriteString(log)
				action := BattleAction{Attacker: attacker.Name, Skill: "普通攻击", Type: "physical"}
				if strings.Contains(log, "暴击") {
					action.Crit = true
				}
				if strings.Contains(log, "闪避") || strings.Contains(log, "miss") {
					action.Dodge = true
				}
				if dmg > 0 {
					dmg = applyDefenseEffects(defender, attacker, dmg, &sb)
					action.Damage = dmg
					if defender == &b {
						b.HP -= dmg
						if b.HP < 0 {
							b.HP = 0
						}
					} else {
						a.HP -= dmg
						if a.HP < 0 {
							a.HP = 0
						}
					}
				}
				roundActions = append(roundActions, action)
			}
		}

		// 处理双方状态效果
		tickLog := a.TickStatuses() + b.TickStatuses()
		if tickLog != "" {
			sb.WriteString(tickLog)
		}

		// 血量展示
		sb.WriteString(fmt.Sprintf("  🔴 %s：❤️ %d/%d", a.Name, a.HP, a.MaxHP))
		if len(a.Statuses) > 0 {
			sb.WriteString(" [")
			for i, s := range a.Statuses {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(s.Name)
			}
			sb.WriteString("]")
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  🔵 %s：❤️ %d/%d", b.Name, b.HP, b.MaxHP))
		if len(b.Statuses) > 0 {
			sb.WriteString(" [")
			for i, s := range b.Statuses {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(s.Name)
			}
			sb.WriteString("]")
		}
		sb.WriteString("\n\n")

		// 记录本回合动画数据
		if ev, ok := eventRounds[round]; ok {
			roundEvent = ev.Name
		}
		allRounds = append(allRounds, BattleRound{Number: round, Actions: roundActions, Event: roundEvent})

		if a.HP <= 0 || b.HP <= 0 {
			break
		}

		// 交换攻防
		attacker, defender = defender, attacker
		round++
	}

	// 超时判定
	if a.HP > 0 && b.HP > 0 {
		sb.WriteString("⏰ 战斗超时！按剩余血量百分比判定胜负\n")
		aRatio := float64(a.HP) / float64(a.MaxHP)
		bRatio := float64(b.HP) / float64(b.MaxHP)
		if aRatio > bRatio {
			b.HP = 0
		} else if bRatio > aRatio {
			a.HP = 0
		} else {
			if rand.Intn(2) == 0 {
				b.HP = 0
			} else {
				a.HP = 0
			}
		}
	}

	// ---- 结果判定 ----
	sb.WriteString("╔══════════════════════╗\n")
	sb.WriteString("║       🏆  战斗结果       ║\n")
	sb.WriteString("╚══════════════════════╝\n")

	var winnerID int
	if a.HP > 0 {
		winnerID = duel.Initiator
		sb.WriteString(fmt.Sprintf("🏆 胜利者：%s！\n", a.Name))
		sb.WriteString(fmt.Sprintf("💀 %s 已倒下（剩余 %d/%d 血）\n\n", b.Name, b.HP, b.MaxHP))
	} else if b.HP > 0 {
		winnerID = duel.Challenger
		sb.WriteString(fmt.Sprintf("🏆 胜利者：%s！\n", b.Name))
		sb.WriteString(fmt.Sprintf("💀 %s 已倒下（剩余 %d/%d 血）\n\n", a.Name, a.HP, a.MaxHP))
	} else {
		// 平局
		sb.WriteString("🤝 平局！双方同归于尽！\n")
		sb.WriteString("💰 积分已退还双方\n")
		// 记录平局
		record.Rounds = allRounds
		record.Winner = ""
		record.Initiator.FinalHP = a.HP
		record.Challenger.FinalHP = b.HP
		db.Model(&User{}).Where("number = ?", duel.Initiator).Update("coin", gorm.Expr("coin + ?", bet))
		db.Model(&User{}).Where("number = ?", duel.Challenger).Update("coin", gorm.Expr("coin + ?", bet))
		RecordCoinLog(duel.Initiator, bet, "游戏", "决斗平局退还", BotContext())
		RecordCoinLog(duel.Challenger, bet, "游戏", "决斗平局退还", BotContext())
		cleanupDuel(duelID)
		sender.Reply(fmt.Sprintf("决斗ID: %s\n%s", duelID, sb.String()))
		return
	}

	// 发放奖励
	reward := bet * 2
	db.Model(&User{}).Where("number = ?", winnerID).Update("coin", gorm.Expr("coin + ?", reward))
	RecordCoinLog(winnerID, reward, "游戏", "决斗获胜奖励", BotContext())
	sb.WriteString(fmt.Sprintf("💰 胜者获得 %d 积分（净赚 %d）\n", reward, bet))

	// 生成动画HTML
	record.Rounds = allRounds
	if a.HP > 0 {
		record.Winner = a.Name
		record.Initiator.FinalHP = a.HP
		record.Challenger.FinalHP = b.HP
	} else {
		record.Winner = b.Name
		record.Initiator.FinalHP = a.HP
		record.Challenger.FinalHP = b.HP
	}
	cleanupDuel(duelID)
	sender.Reply(fmt.Sprintf("决斗ID: %s\n%s", duelID, sb.String()))
}

// applyDefenseEffects 处理护盾、反弹等防御效果，返回最终伤害
func applyDefenseEffects(defender, attacker *BattleAttr, dmg int, sb *strings.Builder) int {
	// 无敌/镜像
	for i, s := range defender.Statuses {
		if s.Type == "shield" {
			if s.Name == "无敌" {
				// 反弹30%
				rebound := dmg * s.Value / 100
				attacker.HP -= rebound
				if attacker.HP < 0 {
					attacker.HP = 0
				}
				sb.WriteString(fmt.Sprintf("  💎 【无敌霸体】触发！完全免伤，反弹 %d 伤害！\n", rebound))
				// 移除状态
				defender.Statuses = append(defender.Statuses[:i], defender.Statuses[i+1:]...)
				return 0
			} else if s.Name == "镜像" {
				// 反弹100%
				attacker.HP -= dmg
				if attacker.HP < 0 {
					attacker.HP = 0
				}
				sb.WriteString(fmt.Sprintf("  🪞 【魔法镜像】触发！伤害完全反弹 %d！\n", dmg))
				defender.Statuses = append(defender.Statuses[:i], defender.Statuses[i+1:]...)
				return 0
			} else if s.Name == "护盾" {
				// 吸收伤害
				absorbed := s.Value
				if absorbed > dmg {
					absorbed = dmg
				}
				dmg -= absorbed
				s.Value -= absorbed
				if s.Value <= 0 {
					defender.Statuses = append(defender.Statuses[:i], defender.Statuses[i+1:]...)
					sb.WriteString(fmt.Sprintf("  🔵 【护盾】吸收 %d 伤害（护盾耗尽）\n", absorbed))
				} else {
					defender.Statuses[i] = s
					sb.WriteString(fmt.Sprintf("  🔵 【护盾】吸收 %d 伤害（剩余 %d）\n", absorbed, s.Value))
				}
				break
			} else if s.Name == "反弹盾" {
				// 反弹50%
				rebound := dmg * 50 / 100
				attacker.HP -= rebound
				if attacker.HP < 0 {
					attacker.HP = 0
				}
				sb.WriteString(fmt.Sprintf("  🔁 【反弹盾】触发！反弹 %d 伤害（50%%）\n", rebound))
				defender.Statuses = append(defender.Statuses[:i], defender.Statuses[i+1:]...)
				break
			}
		}
	}
	return dmg
}

// smartSelectSkill 智能技能选择（考虑血量、CD、对方状态）
func smartSelectSkill(attacker, defender *BattleAttr, round int, cd map[string]int) *BattleSkill {
	// 50%概率使用技能（避免太频繁）
	if rand.Intn(100) >= 55 {
		return nil
	}

	var available []BattleSkill
	for _, sk := range attacker.Skills {
		if cd[sk.Name] <= 0 {
			available = append(available, sk)
		}
	}
	if len(available) == 0 {
		return nil
	}

	hpRatio := float64(attacker.HP) / float64(attacker.MaxHP)

	// 优先级：血量低时优先治疗/防御；正常时选攻击；对方快死时选爆发
	// 1. 濒死时优先续命
	if hpRatio < 0.25 {
		for i := range available {
			if available[i].Type == "heal" {
				return &available[i]
			}
		}
		for i := range available {
			if available[i].Type == "defense" {
				return &available[i]
			}
		}
	}

	// 2. 对方血量很低时优先爆发
	defRatio := float64(defender.HP) / float64(defender.MaxHP)
	if defRatio < 0.3 {
		for _, priority := range []string{"致命打击", "赌命一搏", "天降鸿运", "元素风暴", "复仇之刃"} {
			for i := range available {
				if available[i].Name == priority {
					return &available[i]
				}
			}
		}
	}

	// 3. 中等血量时随机使用技能
	idx := rand.Intn(len(available))
	return &available[idx]
}

// cleanupDuel 清理决斗房间
func cleanupDuel(duelID string) {
	delete(duels, duelID)
	for i, d := range duelsByTime {
		if d.ID == duelID {
			duelsByTime = append(duelsByTime[:i], duelsByTime[i+1:]...)
			break
		}
	}
}

// ClassListMsg 返回职业选择提示消息
func ClassListMsg() string {
	var sb strings.Builder
	sb.WriteString("🎮 请选择你的职业（回复数字序号）：\n\n")
	for i, c := range AllClasses {
		sb.WriteString(fmt.Sprintf("【%d】%s %s\n", i+1, c.Emoji+string(c.Name), c.Description))
	}
	sb.WriteString("\n30秒内选择，超时将随机分配职业")
	return sb.String()
}

// ---- 保留旧接口兼容（已废弃，不再使用） ----
type Attributes = BattleAttr
type Skill = BattleSkill
