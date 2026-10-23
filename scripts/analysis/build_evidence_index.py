"""Static evidence indexing only. Never imports or executes archive code."""
from pathlib import Path
import csv, json, re, collections, zipfile, hashlib

P = Path(__file__).resolve().parents[2]
ROOT = P / 'analysis/extracted'
OUT = P / 'analysis'
rows = list(csv.DictReader((OUT/'FILE_INVENTORY.csv').open(encoding='utf-8-sig')))
functions, texts, nested, modules, headers, ui = [], [], [], {}, collections.Counter(), []
for row in rows:
    path = row['path']; f = ROOT/path; lo = path.lower()
    categories = set(re.split(r'[;|]', row['categories'])) - {'未知资源'}
    ext = f.suffix.lower()
    if ext in {'.java','.smali','.dex','.so'}: categories.add('原生/SDK/反编译')
    if ext in {'.md','.py','.csv','.txt','.yml'}: categories.add('恢复工具/文档')
    if ext in {'.png','.jpg','.jpeg'}:
        categories.add('图片'); headers[row['header_hex'][:16]] += 1
    if ext == '.plist': categories.add('图集/动画')
    for terms, cat in [(['cmd_','frame','socket','gateway'],'网络通信'),(['logon','login','register'],'登录和账号'),(['teahouse','th_'],'茶馆系统'),(['battle','room'],'房间/战绩'),(['score','gameend'],'战绩和结算'),(['gamelogic','checkhu','hu_table'],'游戏规则'),(['gameclientengine'],'游戏逻辑'),(['config','dfex','filemd5'],'游戏配置'),(['userinfo','modifyinfo','face'],'用户资料'),(['layer/','layer\\','view.lua'],'UI布局')]:
        if any(x in lo for x in terms): categories.add(cat)
    row['categories'] = ';'.join(sorted(categories or {'未知资源'}))
    if ext == '.zip':
        with zipfile.ZipFile(f) as z:
            entries = [i.filename for i in z.infolist() if not i.is_dir()]
            nested.append({'path':path,'sha256':row['sha256'],'files':len(entries),'crc_bad':z.testzip(),'platehes_matches':[s for s in entries if re.search(r'platehes|hongershi|红二十',s,re.I)], 'modules': sorted(set(m.group(1) for s in entries if (m:=re.search(r'(?:chess|recreational)/([^/]+)/',s))))})
    if '/work/decoded/' not in lo or ext != '.lua': continue
    content = f.read_text(encoding='utf-8',errors='replace')
    if (m:=re.search(r'/game/game/(?:chess|recreational)/([^/]+)/',lo)):
        modules[m.group(1)] = modules.get(m.group(1),0)+1
    for n,line in enumerate(content.splitlines(),1):
        # Index names, no source-code reuse or original endpoint/credential export.
        if m:=re.search(r'^\s*(?:local\s+)?function\s+([\w.:]+)',line): functions.append({'path':path,'line':n,'function':m.group(1)})
        if re.search(r'断[勾掐卡]|红中|红二十|斗十四|跑得快|platehes',line) and not re.search(r'key|secret|password|https?://|token',line,re.I):
            texts.append({'path':path,'line':n,'text':line.strip()[:240]})
    if '/views/' in lo or '/layer/' in lo:
        ui.append({'path':path,'resources':sorted(set(re.findall(r'["\']([^"\'\n]+\.(?:png|jpg|plist|mp3|wav))["\']',content))), 'constructors':[x for x in functions if x['path']==path and re.search(r'ctor|onCreate|onTouch|onClick',x['function'])]})
with (OUT/'FILE_INVENTORY.csv').open('w',encoding='utf-8-sig',newline='') as o:
    w=csv.DictWriter(o,fieldnames=rows[0].keys());w.writeheader();w.writerows(rows)
idx=json.loads((OUT/'RESOURCE_INDEX.json').read_text(encoding='utf-8'))
idx['semantic_index_version']=2
idx['files']=rows
(OUT/'RESOURCE_INDEX.json').write_text(json.dumps(idx,ensure_ascii=False,indent=2),encoding='utf-8')
for name,value in [('FUNCTION_INDEX',functions),('GAME_TEXT_EVIDENCE',texts),('NESTED_ARCHIVES',nested),('UI_RESOURCE_REFERENCES',ui)]:
    (OUT/(name+'.json')).write_text(json.dumps(value,ensure_ascii=False,indent=2),encoding='utf-8')
summary={'modules':modules,'decoded_functions':len(functions),'game_text_lines':len(texts),'ui_files':len(ui),'image_headers':headers.most_common(12),'nested_archives':nested,'all_filename_platehes_matches':[r['path'] for r in rows if re.search(r'platehes|hongershi|红二十',r['path'],re.I)],'category_counts':collections.Counter(c for r in rows for c in r['categories'].split(';'))}
(OUT/'SEMANTIC_SCAN.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2),encoding='utf-8')
print(json.dumps(summary,ensure_ascii=False,indent=2))
