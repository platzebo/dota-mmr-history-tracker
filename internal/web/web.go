package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/platzebo/dota-mmr-history-tracker/internal/ledger"
	"github.com/platzebo/dota-mmr-history-tracker/internal/store"
)

type Server struct{ store *store.Store }

func New(s *store.Store) *Server { return &Server{store: s} }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/api/matches", s.matches)
	mux.HandleFunc("/api/summary", s.summary)
	mux.HandleFunc("/export.csv", s.csv)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

type matchView struct {
	ledger.Record
	HeroName string `json:"hero_name"`
}

func addHeroNames(rows []ledger.Record) []matchView {
	out := make([]matchView, 0, len(rows))
	for _, row := range rows {
		out = append(out, matchView{Record: row, HeroName: ledger.HeroName(row.HeroID)})
	}
	return out
}

func (s *Server) matches(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListMatches()
	if err != nil {
		writeJSON(w, nil, err)
		return
	}
	writeJSON(w, addHeroNames(rows), nil)
}
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListMatches()
	if err != nil {
		writeJSON(w, nil, err)
		return
	}
	writeJSON(w, ledger.Summarize(rows), nil)
}
func (s *Server) csv(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListMatches()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out, err := ledger.ExportCSV(rows)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "text/csv; charset=utf-8")
	w.Header().Set("content-disposition", "attachment; filename=dota-mmr-history-tracker.csv")
	fmt.Fprint(w, out)
}
func writeJSON(w http.ResponseWriter, v any, err error) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	if err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

const page = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Dota MMR History Tracker</title>
<style>
:root{color-scheme:dark;--bg:#070b14;--panel:#0f172a;--panel2:#111c33;--line:#263449;--muted:#94a3b8;--text:#e5e7eb;--blue:#60a5fa;--cyan:#22d3ee;--green:#86efac;--red:#fca5a5;--yellow:#facc15;--purple:#c084fc}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 25% 0,#14213d 0,#070b14 42rem);color:var(--text);font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}.wrap{max-width:1440px;margin:0 auto;padding:24px}header{display:flex;justify-content:space-between;gap:18px;align-items:flex-end;margin-bottom:20px}.eyebrow{color:var(--cyan);font-weight:700;letter-spacing:.12em;text-transform:uppercase;font-size:12px}h1{font-size:clamp(30px,4vw,56px);line-height:1;margin:.2rem 0 .5rem}.sub{color:var(--muted);max-width:860px}.actions{display:flex;gap:10px;flex-wrap:wrap}.btn{background:#172554;border:1px solid #1d4ed8;color:#dbeafe;text-decoration:none;border-radius:12px;padding:10px 14px;font-weight:700}.btn.secondary{background:#111827;border-color:#334155}.grid{display:grid;gap:16px}.cards{grid-template-columns:repeat(7,minmax(120px,1fr));margin-bottom:16px}.card,.panel{background:linear-gradient(180deg,rgba(17,28,51,.96),rgba(15,23,42,.96));border:1px solid var(--line);box-shadow:0 18px 45px rgba(0,0,0,.25);border-radius:18px}.card{padding:16px}.card small{display:block;color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.08em}.card strong{display:block;font-size:26px;margin-top:8px}.card .delta{font-size:13px;margin-top:4px}.main{grid-template-columns:minmax(0,2fr) minmax(300px,.85fr);align-items:start}.panel{padding:18px}.panel h2{margin:.1rem 0 1rem;font-size:18px}.toolbar{display:flex;gap:12px;flex-wrap:wrap;align-items:center;margin-bottom:12px;color:var(--muted)}select,input,button{background:#0b1220;color:var(--text);border:1px solid #334155;border-radius:10px;padding:8px 10px}button{cursor:pointer}.chartBox{position:relative;height:520px;border-radius:18px;background:linear-gradient(180deg,#0b1220,#08111f);border:1px solid #1f2a3d;overflow:hidden}.pointerLine{position:absolute;top:0;bottom:0;width:1px;background:linear-gradient(180deg,transparent,#93c5fd,transparent);display:none;pointer-events:none;opacity:.85}canvas{width:100%;height:100%;display:block}.tooltip{position:absolute;pointer-events:none;background:#020617;border:1px solid #475569;color:#e2e8f0;border-radius:12px;padding:10px;font-size:13px;box-shadow:0 10px 30px rgba(0,0,0,.45);display:none;white-space:nowrap}.legend{display:flex;gap:16px;color:var(--muted);font-size:13px;margin-top:10px;flex-wrap:wrap}.dot{display:inline-block;width:10px;height:10px;border-radius:99px;margin-right:6px}.two{grid-template-columns:1fr 1fr;margin-top:16px}.heroList{display:grid;gap:8px}.heroRow{display:grid;grid-template-columns:minmax(128px,1fr) 1fr 70px;gap:10px;align-items:center;color:#cbd5e1}.bar{height:8px;background:#1f2937;border-radius:999px;overflow:hidden}.bar>i{display:block;height:100%;border-radius:999px}.tableWrap{overflow:auto;max-height:560px}table{border-collapse:collapse;width:100%;font-size:14px}th,td{padding:10px 12px;border-bottom:1px solid #1f2937;text-align:left;white-space:nowrap}th{position:sticky;top:0;background:#101a2e;color:#93c5fd;z-index:1}.pos{color:var(--green)}.neg{color:var(--red)}.muted{color:var(--muted)}.empty{padding:48px;text-align:center;color:var(--muted)}@media(max-width:1050px){.cards{grid-template-columns:repeat(2,1fr)}.main,.two{grid-template-columns:1fr}header{display:block}.chartBox{height:420px}}@media(max-width:620px){.wrap{padding:14px}.cards{grid-template-columns:1fr}.chartBox{height:360px}}
</style>
</head>
<body>
<div class="wrap">
<header><div><div class="eyebrow">Local-first · Steam GC · no manual MMR entry</div><h1>Dota MMR History Tracker</h1><div class="sub">Automatic ranked MMR history reconstructed from Dota GameCoordinator <code>previous_rank</code> + <code>rank_change</code>. Data stays in your local SQLite database.</div></div><div class="actions"><a class="btn" href="/export.csv">Export CSV</a><a class="btn secondary" href="/api/matches">Raw JSON</a></div></header>
<section class="cards grid" id="cards"></section>
<section class="main grid"><div class="panel"><div class="toolbar"><strong>MMR timeline</strong><label>Range <select id="range"><option value="all">All</option><option value="30">Last 30</option><option value="100">Last 100</option><option value="250">Last 250</option><option value="500">Last 500</option></select></label><button id="resetZoom">Reset zoom</button><span id="status" class="muted"></span><span id="hoverInfo" class="muted"></span></div><div class="chartBox"><canvas id="chart"></canvas><div class="pointerLine" id="pointerLine"></div><div class="tooltip" id="tip"></div></div><div class="legend"><span><i class="dot" style="background:var(--blue)"></i>MMR</span><span><i class="dot" style="background:var(--green)"></i>Wins / positive Δ</span><span><i class="dot" style="background:var(--red)"></i>Losses / negative Δ</span><span>Zoom selection: use the Range dropdown</span></div></div><aside class="panel"><h2>MMR per Hero</h2><div id="heroes" class="heroList"></div></aside></section>
<section class="two grid"><div class="panel"><h2>Recent matches</h2><div class="tableWrap"><table><thead><tr><th>Date</th><th>Match</th><th>Hero</th><th>Before</th><th>Δ</th><th>After</th><th>Solo</th></tr></thead><tbody id="rows"></tbody></table></div></div><div class="panel"><h2>Sync tips</h2><p class="muted"><strong>Auto sync:</strong> run <code>sync --auto --qr</code> for normal updates. The tool starts at the newest Dota GC page and stops when it reaches a match ID that is already in your database.</p><p class="muted"><strong>Backfill:</strong> use <code>sync --skip-pages N</code> for older ranges. One page is 20 GC rows; <code>--skip-pages 200</code> starts after roughly the newest 4,000 history rows.</p><p class="muted">Keep Dota 2 closed while syncing and leave the default <code>--page-delay 1s</code> enabled for conservative GameCoordinator pacing.</p></div></section>
</div>
<script>
let allRows=[], viewRows=[]; const fmt=n=>Number(n||0).toLocaleString(); const fmtDate=t=>new Date(t*1000).toLocaleDateString(undefined,{year:'2-digit',month:'2-digit',day:'2-digit'}); const fmtDateTime=t=>new Date(t*1000).toLocaleString();
function cls(v){return v>=0?'pos':'neg'} function signed(v){return (v>0?'+':'')+fmt(v)} function heroLabel(r){return r.hero_name || ('#'+r.hero_id)}
function cards(summary,rows){let last=rows[rows.length-1], first=rows[0], recent=rows.slice(-30).reduce((a,r)=>a+r.rank_change,0); let win=rows.filter(r=>r.rank_change>0).length; let wr=rows.length?Math.round(win/rows.length*100):0; let data=[['Current',summary.current_mmr,'plain'],['Peak',summary.peak_mmr,'plain'],['Lowest',summary.lowest_mmr,'plain'],['Total Δ',summary.total_change,'signed'],['Last 30 Δ',recent,'signed'],['Win %',wr+'%','plain'],['Matches',summary.match_count,'plain']]; document.getElementById('cards').innerHTML=data.map(x=>{let numeric=typeof x[1]==='number', signedMode=x[2]==='signed'; let value=numeric?(signedMode?signed(x[1]):fmt(x[1])):x[1]; let color=numeric?(signedMode?cls(x[1]):''):''; return '<div class="card"><small>'+x[0]+'</small><strong class="'+color+'">'+value+'</strong><div class="delta muted">'+(last&&x[0]==='Current'?'since '+fmtDate(first.start_time):'')+'</div></div>'}).join('')}
function setupCanvas(canvas){let dpr=window.devicePixelRatio||1, rect=canvas.getBoundingClientRect(); canvas.width=Math.max(1,Math.floor(rect.width*dpr)); canvas.height=Math.max(1,Math.floor(rect.height*dpr)); let ctx=canvas.getContext('2d'); ctx.setTransform(dpr,0,0,dpr,0,0); return {ctx,w:rect.width,h:rect.height}}
function drawChart(rows){viewRows=rows; const canvas=document.getElementById('chart'), tip=document.getElementById('tip'); let g=setupCanvas(canvas), ctx=g.ctx,w=g.w,h=g.h; ctx.clearRect(0,0,w,h); if(!rows.length){ctx.fillStyle='#94a3b8';ctx.font='16px system-ui';ctx.fillText('No MMR rows yet. Run sync first.',28,44);return} let pad={l:64,r:20,t:26,b:54}; let vals=rows.flatMap(r=>[r.mmr_before,r.mmr_after]); let min=Math.min(...vals), max=Math.max(...vals); let span=Math.max(1,max-min); min=Math.floor((min-span*.08)/50)*50; max=Math.ceil((max+span*.08)/50)*50; let x=i=>pad.l+i*Math.max(1,(w-pad.l-pad.r)/Math.max(1,rows.length-1)); let y=v=>h-pad.b-((v-min)/Math.max(1,max-min))*(h-pad.t-pad.b); ctx.strokeStyle='#1f2a3d';ctx.lineWidth=1;ctx.fillStyle='#94a3b8';ctx.font='12px system-ui'; for(let k=0;k<=5;k++){let yy=pad.t+k*(h-pad.t-pad.b)/5;let val=Math.round(max-k*(max-min)/5);ctx.beginPath();ctx.moveTo(pad.l,yy);ctx.lineTo(w-pad.r,yy);ctx.stroke();ctx.fillText(fmt(val),8,yy+4)} ctx.strokeStyle='rgba(148,163,184,.25)';ctx.beginPath();ctx.moveTo(pad.l,y(0));ctx.lineTo(w-pad.r,y(0));ctx.stroke(); rows.forEach((r,i)=>{let xx=x(i);ctx.strokeStyle=r.rank_change>=0?'rgba(134,239,172,.38)':'rgba(252,165,165,.38)';ctx.beginPath();ctx.moveTo(xx,y(r.mmr_before));ctx.lineTo(xx,y(r.mmr_after));ctx.stroke()}); ctx.strokeStyle='#60a5fa';ctx.lineWidth=3;ctx.beginPath();rows.forEach((r,i)=>{let xx=x(i),yy=y(r.mmr_after);i?ctx.lineTo(xx,yy):ctx.moveTo(xx,yy)});ctx.stroke(); rows.forEach((r,i)=>{ctx.fillStyle=r.rank_change>=0?'#86efac':'#fca5a5';ctx.beginPath();ctx.arc(x(i),y(r.mmr_after),2.5,0,Math.PI*2);ctx.fill()}); ctx.fillStyle='#94a3b8';ctx.fillText(fmtDate(rows[0].start_time),pad.l,h-18);ctx.fillText(fmtDate(rows[rows.length-1].start_time),Math.max(pad.l,w-130),h-18); canvas.onmousemove=e=>{let rect=canvas.getBoundingClientRect(),mx=e.clientX-rect.left;let idx=Math.round((mx-pad.l)/Math.max(1,(w-pad.l-pad.r)/Math.max(1,rows.length-1)));idx=Math.max(0,Math.min(rows.length-1,idx));let r=rows[idx];tip.style.display='block';tip.style.left=Math.min(w-230,Math.max(8,x(idx)+12))+'px';tip.style.top=Math.max(8,y(r.mmr_after)-58)+'px';tip.innerHTML='<b>'+fmtDateTime(r.start_time)+'</b><br>MMR '+fmt(r.mmr_before)+' → '+fmt(r.mmr_after)+' <span class="'+cls(r.rank_change)+'">('+signed(r.rank_change)+')</span><br>Match '+r.match_id+' · '+heroLabel(r)}; canvas.onmouseleave=()=>tip.style.display='none'}
function heroStats(rows){let m=new Map();rows.forEach(r=>{let name=heroLabel(r);let v=m.get(name)||{hero:name,delta:0,matches:0};v.delta+=r.rank_change;v.matches++;m.set(name,v)});let arr=[...m.values()].sort((a,b)=>Math.abs(b.delta)-Math.abs(a.delta)).slice(0,12);let max=Math.max(1,...arr.map(x=>Math.abs(x.delta)));document.getElementById('heroes').innerHTML=arr.length?arr.map(h=>'<div class="heroRow"><b>'+h.hero+'</b><div><div class="bar"><i style="width:'+(Math.abs(h.delta)/max*100)+'%;background:'+(h.delta>=0?'var(--green)':'var(--red)')+'"></i></div><small class="muted">'+h.matches+' matches</small></div><span class="'+cls(h.delta)+'">'+signed(h.delta)+'</span></div>').join(''):'<div class="empty">No hero stats yet</div>'}
function table(rows){document.getElementById('rows').innerHTML=rows.slice().reverse().slice(0,250).map(r=>'<tr><td>'+fmtDateTime(r.start_time)+'</td><td>'+r.match_id+'</td><td>'+heroLabel(r)+'</td><td>'+fmt(r.mmr_before)+'</td><td class="'+cls(r.rank_change)+'">'+signed(r.rank_change)+'</td><td>'+fmt(r.mmr_after)+'</td><td>'+(r.solo_rank?'yes':'no')+'</td></tr>').join('')}
function apply(){let n=document.getElementById('range').value;let rows=n==='all'?allRows:allRows.slice(-Number(n));document.getElementById('status').textContent=rows.length+' shown / '+allRows.length+' total';drawChart(rows);heroStats(rows);table(rows)}
Promise.all([fetch('/api/summary').then(r=>r.json()),fetch('/api/matches').then(r=>r.json())]).then(([s,rows])=>{allRows=(rows||[]).slice().sort((a,b)=>a.start_time-b.start_time||a.match_id-b.match_id);cards(s,allRows);apply();document.getElementById('range').onchange=apply;document.getElementById('resetZoom').onclick=()=>{document.getElementById('range').value='all';apply()};window.addEventListener('resize',apply)}).catch(err=>{document.body.innerHTML='<div class="wrap"><div class="panel"><h1>Failed to load data</h1><pre>'+err+'</pre></div></div>'});
</script>
</body>
</html>`
