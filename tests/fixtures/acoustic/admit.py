# SPDX-License-Identifier: Apache-2.0
# Reproduce from the exact official archive; never run an acoustic model.
import hashlib,json,re,sys,tarfile
from pathlib import Path
a=Path(sys.argv[1]);out=Path(sys.argv[2]);assert hashlib.file_digest(a.open("rb"),"sha256").hexdigest()=="76f87d090650617fca0cac8f88b9416e0ebf80350acb97b343a85fa903728ab3"
rows={s:[] for s in ("84","174","251")};texts={};metadata={}
with tarfile.open(a,"r|gz") as t:
 for m in t:
  if not m.isfile():continue
  p=m.name.split("/");stem=Path(m.name).stem
  if len(p)>=5 and p[-3] in rows and m.name.endswith(".flac"):
   f=t.extractfile(m);h=f.read(42);assert h[:4]==b"fLaC" and h[4]&127==0 and int.from_bytes(h[5:8],"big")==34
   b=int.from_bytes(h[18:26],"big");rate=b>>44;n=b&((1<<36)-1);assert rate==16000
   if rate*6<=n<=rate*20:rows[p[-3]].append({"id":stem,"path":stem+".flac","size_bytes":m.size,"samples":n,"sample_rate":rate,"speaker":p[-3],"archive_member":m.name})
  elif m.name.endswith(".trans.txt") and len(p)>=4 and p[-3] in rows:
   for l in t.extractfile(m).read().decode().splitlines():
    k,v=l.split(" ",1);texts[k]=v
  elif p[-1] in ("SPEAKERS.TXT","CHAPTERS.TXT","LICENSE.TXT","README.TXT"):
   assert m.size<2000000;metadata[p[-1]]=t.extractfile(m).read()
key=lambda r:tuple(map(int,r["id"].split("-")))
selected=[]
for speaker in ("84","174"):
 candidates=sorted(rows[speaker],key=key);assert len(candidates)>1
 enroll=candidates[0];held=next((x for x in candidates if x["id"].split("-")[1]!=enroll["id"].split("-")[1]),candidates[1])
 selected.extend([{**enroll,"role":"enrollment"},{**held,"role":"held-out"}])
selected.append({**sorted(rows["251"],key=key)[0],"role":"unseen"})
for r in selected:r["transcript"]=texts[r["id"]]
assert not out.exists();out.mkdir(parents=True)
byname={r["archive_member"]:r for r in selected}
with tarfile.open(a,"r|gz") as t:
 for m in t:
  if m.name in byname:
   assert m.isfile();r=byname[m.name];data=t.extractfile(m).read();assert len(data)==r["size_bytes"];r["sha256"]=hashlib.sha256(data).hexdigest();(out/r["path"]).write_bytes(data)
references="".join(r["id"]+" "+r["transcript"]+"\n" for r in selected).encode();(out/"references.txt").write_bytes(references)
(out/"LICENSE-UPSTREAM.txt").write_bytes(metadata.get("LICENSE.TXT",b"CC BY 4.0: https://creativecommons.org/licenses/by/4.0/\n"))
parts=[]
chapters={r["id"].split("-")[1] for r in selected}
for name in ("SPEAKERS.TXT","CHAPTERS.TXT"):
 wanted=set(rows) if name=="SPEAKERS.TXT" else chapters
 for l in metadata[name].decode().splitlines():
  if l.startswith(";") or l.split("|",1)[0].strip() in wanted:parts.append(l)
(out/"source-metadata.txt").write_text("\n".join(parts)+"\n",encoding="utf-8",newline="\n")
manifest={"format":"insonic-acoustic-reference-v1","source":{"url":"https://www.openslr.org/resources/12/dev-clean.tar.gz","size_bytes":a.stat().st_size,"sha256":"76f87d090650617fca0cac8f88b9416e0ebf80350acb97b343a85fa903728ab3","md5":"42e2234ba48799c1f50f24a7926300a1","license":"CC-BY-4.0","license_url":"https://creativecommons.org/licenses/by/4.0/","authors":["Vassil Panayotov","Guoguo Chen","Daniel Povey","Sanjeev Khudanpur"]},"selection":{"enrolled_speakers":["84","174"],"unseen_speaker":"251","min_samples":96000,"max_samples":320000,"policy":"ascending numeric reader/chapter/utterance; first eligible enrollment, first eligible different chapter held-out when available, otherwise next distinct utterance; frozen before inference"},"references":{"path":"references.txt","size_bytes":len(references),"sha256":hashlib.sha256(references).hexdigest()},"clips":selected,"sequence":{"clips":[r["id"] for r in selected if r["role"]=="held-out"],"gap_samples":8000},"matching":{"threshold":0.75,"ambiguity_margin":0.1,"min_evidence_us":1000000}}
(out/"manifest.json").write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+"\n",encoding="utf-8",newline="\n")
print(json.dumps({"clips":[{k:r[k] for k in ("id","speaker","role","samples","size_bytes")} for r in selected],"bytes":sum(r["size_bytes"] for r in selected),"metadata":list(metadata)}))
