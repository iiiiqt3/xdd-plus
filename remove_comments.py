import re
import os

go_files = []
for root, dirs, files in os.walk('.'):
    for f in files:
        if f.endswith('.go'):
            go_files.append(os.path.join(root, f))

for filename in go_files:
    with open(filename, 'r') as f:
        lines = f.readlines()

    new_lines = []
    for line in lines:
        stripped = line.strip()
        if stripped.startswith('//'):
            rest = stripped[2:].strip()
            # Keep Chinese comments
            if any('\u4e00' <= c <= '\u9fff' for c in rest):
                new_lines.append(line)
                continue
            # Keep section separators
            if rest.startswith('===') or rest.startswith('---'):
                new_lines.append(line)
                continue
            # Keep TODO/FIXME/NOTE etc.
            if re.match(r'^(TODO|FIXME|NOTE|HACK|XXX|BUG|OPTIMIZE|REVIEW|DEPRECATED)\b', rest):
                new_lines.append(line)
                continue
            # Keep go:embed and go:generate directives
            if rest.startswith('go:'):
                new_lines.append(line)
                continue
            # Remove lines that are clearly commented-out code
            code_patterns = [
                r'^(func|var |if |for |return |type |case |switch |go |defer |break |continue )',
                r'^(cks|ck |cookie|sender)',
                r'^(db\.|Config\.|GetJd|handleJd|SendQQ|SendTG|SendWx)',
                r'^(Reply|Update|Where|First|Create|Model|Pluck|Expr)',
                r'^(Join|Format|Sprintf|Println|Printf)',
                r'^(time\.|rand\.|strconv\.|fmt\.|strings\.)',
                r'^(os\.|io\.|net\.|http\.|json\.|regexp\.)',
                r'^(sort\.|sync\.|context\.|errors\.|log\.)',
                r'^(path\.|filepath\.|ioutil\.|bufio\.|bytes\.|encoding)',
                r'^(rsp\.|req\.|resp\.|msg\.|data\.|result\.)',
                r'^(err\.|u\.|nck\.|bot\.|self\.|friedns)',
                r'^(defer |close\()',
                r'^(if |for |switch |select |case |default:)',
                r'^(\{|\}|\)|else|else if)',
                r'^(//|/\*)',
                r'^(##|->|=>)',
                r'^(log\.|logs\.)',
                r'^(\d+)',
                r'^(//\s*$)',
                r'^(cookieUpdate\.|UpdateAt|result\d)',
                r'^(to\.|asset\.|bean\.)',
                r'^(zjb|jxzz|jdz|jingxiangzhi)',
                r'^(AutoAddcoin|smslogin|smsLogin)',
                r'^(smsLogin|smslogin)',
            ]
            is_code = False
            for pattern in code_patterns:
                if re.match(pattern, rest):
                    is_code = True
                    break
            if is_code:
                continue
            if re.match(r'^[a-zA-Z_][\w.]*\s*[=:(]', rest):
                continue
            new_lines.append(line)
        else:
            new_lines.append(line)

    with open(filename, 'w') as f:
        f.writelines(new_lines)

    print(f'Done: {filename}')