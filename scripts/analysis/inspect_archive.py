"""Read-only archive analysis. Never executes archive contents or overwrites files."""
from __future__ import annotations
import collections, csv, hashlib, json, os, re, shutil, stat, sys, time, zipfile
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[2]
ARCHIVE = Path(r'C:\Users\xinha\Downloads\majiang.zip')
OUT = ROOT / 'analysis'
DEST = OUT / 'extracted'

def sha256(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for b in iter(lambda: f.read(1024 * 1024), b''): h.update(b)
    return h.hexdigest()

def safe_path(name, root):
    n = name.replace('\\', '/')
    p = PurePosixPath(n)
    if p.is_absolute() or any(x in ('..', '') for x in p.parts) or ':' in n or '\x00' in n:
        raise ValueError(f'Unsafe member: {name!r}')
    for part in p.parts:
        if part.rstrip(' .') != part or part.split('.')[0].upper() in {'CON','PRN','AUX','NUL',*(f'COM{i}' for i in range(1,10)),*(f'LPT{i}' for i in range(1,10))}:
            raise ValueError(f'Unsafe Windows member: {name!r}')
    result = root.joinpath(*p.parts).resolve()
    if not result.is_relative_to(root.resolve()): raise ValueError(f'Escape: {name!r}')
    return result

def magic_type(head):
    for prefix, typ in [(b'\x1bLua','Lua bytecode'),(b'\x1bLJ','LuaJIT bytecode'),(b'\x7fELF','ELF'),(b'dex\n','DEX'),(b'PK\x03\x04','ZIP'),(b'\x89PNG','PNG'),(b'\xff\xd8\xff','JPEG'),(b'RIFF','RIFF/WAV'),(b'OggS','OGG'),(b'ID3','MP3'),(b'\x1f\x8b','GZIP'),(b'CCZ!','CCZ'),(b'PVR\x03','PVR')]:
        if head.startswith(prefix): return typ
    if not head: return 'empty'
    if b'\x00' in head: return 'binary/unknown'
    try: head.decode('utf-8'); return 'text/utf8'
    except UnicodeDecodeError: return 'binary/unknown'

def categories(path, kind):
    s = path.lower(); c = []
    patterns = {
        '游戏逻辑':r'/(chess|recreational)/|gamelogic|gameframe|gamemain',
        '游戏规则':r'rule|gamelogic|gamehelp|help.*game',
        '游戏配置':r'config|define|cmd|setting',
        '网络通信':r'net|socket|protocol|message|cmd|packet|tcp',
        '登录和账号':r'login|account|register|logon',
        '房间系统':r'room|table|chair|create|join',
        '茶馆系统':r'tea|club|group|guild|union|circle',
        '用户资料':r'user|profile|head|personal',
        '战绩和结算':r'record|replay|result|settle|endex|gameend|score',
        '图片':r'\.(png|jpg|jpeg|webp)$',
        '图集':r'\.(plist|atlas|pvr|ccz)$',
        '动画':r'anim|action|skeleton|spine|\.csb$|\.csi$',
        '音效':r'\.(wav|mp3|ogg)$',
        '字体':r'font|\.(ttf|otf|fnt)$',
        'UI 布局':r'cocostudio|layout|scene|layer|ui/|\.csb$|\.csi$|\.csd$|\.plist$',
        '多语言文本':r'lang|localiz|strings\.xml|text|help',
    }
    for k, pattern in patterns.items():
        if re.search(pattern, s): c.append(k)
    return c or ['未知资源']

def main():
    OUT.mkdir(parents=True, exist_ok=True)
    if DEST.exists(): raise SystemExit('Refusing overwrite: analysis/extracted already exists')
    start = time.monotonic()
    archive_hash = sha256(ARCHIVE)
    with zipfile.ZipFile(ARCHIVE) as z:
        infos = z.infolist(); files = [i for i in infos if not i.is_dir()]
        total = sum(i.file_size for i in files)
        if total > 4 * 1024**3 or total > shutil.disk_usage(OUT).free // 2: raise ValueError('Extraction size limit')
        seen = set()
        for i in infos:
            target = safe_path(i.filename, DEST)
            key = str(target).casefold()
            if key in seen: raise ValueError(f'Duplicate/case collision: {i.filename}')
            seen.add(key)
            if stat.S_ISLNK(i.external_attr >> 16): raise ValueError('Symlink refused')
            if i.flag_bits & 1: raise ValueError('Encrypted outer member')
            if i.file_size > 1024**3 or i.file_size / max(i.compress_size,1) > 10000: raise ValueError('Suspicious compression')
        # Streaming every member below verifies its CRC; write with exclusive creation.
        rows = []
        for i in files:
            path = safe_path(i.filename, DEST); path.parent.mkdir(parents=True, exist_ok=True)
            h = hashlib.sha256(); head = b''
            with z.open(i) as src, path.open('xb') as dst:
                for b in iter(lambda: src.read(1024*1024),b''):
                    if not head: head = b[:256]
                    h.update(b); dst.write(b)
            kind = magic_type(head)
            rows.append(dict(path=i.filename,size=i.file_size,compressed_size=i.compress_size,crc32=f'{i.CRC:08x}',sha256=h.hexdigest(),extension=path.suffix.lower() or '<none>',magic=kind,header_hex=head[:16].hex(),categories=categories(i.filename,kind)))
    with (OUT/'FILE_INVENTORY.csv').open('w',newline='',encoding='utf-8-sig') as f:
        w=csv.DictWriter(f,fieldnames=rows[0].keys());w.writeheader()
        for r in rows: w.writerow({**r,'categories':'|'.join(r['categories'])})
    report=dict(archive=str(ARCHIVE),size_bytes=ARCHIVE.stat().st_size,sha256=archive_hash,file_count=len(rows),uncompressed_bytes=total,crc='PASS: all members streamed with ZIP CRC validation',unsafe_paths=0,encrypted_outer_members=0,extensions=dict(collections.Counter(r['extension'] for r in rows)),magic=dict(collections.Counter(r['magic'] for r in rows)),categories=dict(collections.Counter(c for r in rows for c in r['categories'])),elapsed_seconds=round(time.monotonic()-start,2))
    (OUT/'ARCHIVE_REPORT.json').write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
    (OUT/'RESOURCE_INDEX.json').write_text(json.dumps({'method':'Path/magic classification; multi-label candidates require semantic verification. No protected content copied to production.','summary':report,'files':rows},ensure_ascii=False,indent=2),encoding='utf-8')
    dirs=collections.Counter('/'.join(r['path'].split('/')[:5]) for r in rows)
    (OUT/'DIRECTORY_TREE.txt').write_text('\n'.join(f'{p} [{n} files]' for p,n in sorted(dirs.items())),encoding='utf-8')
    print(json.dumps(report,ensure_ascii=True))

if __name__ == '__main__': main()
