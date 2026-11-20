#!/usr/bin/env node
// build-knowledge-graph.mjs — extrator determinístico (lista fechada de parsers)
// da skill gm-knowledge-graph (CORE do kit golmet — IMP-008). Gera o grafo tipado
// da base de conhecimento (stores T1-T5 do knowledge-routing) e emite 3 artefatos:
//   graph.json · knowledge-graph.html (self-contained, file:// offline) · knowledge-graph.md
// Fontes (read-only; ausente = aviso, EXCETO registry.yaml que ABORTA):
//   T5: <config-root>/registry.yaml (OBRIGATÓRIO) + AGENTS.md (índice lazy) + skills/agents (fundado_em)
//   T1: <packs-root>/domains/INDEX.md (staleness por proxima_revisao)
//   T3: <packs-root>/rules/*.md
//   T2: <packs-root>/accumulation/patterns/INDEX.md
//   T4: <projects-root>/<Produto>/{REQUIREMENTS.md, DECISIONS.md, quality/TRACEABILITY.md,
//       delivery/**/RN-NNN — <título> — Feature.md (fallback Tasks.md, IMP-092), specs/*.md}
// Contrato (SKILL.md): strict SÓ no registry; saída integral ou nada (escreve só após
// montar tudo em memória); --scope delta; --max-nodes (default 3500, poda RN/PEND do
// modo global primeiro, meta.podado); --from-json (regeneração de grafo congelado).
// Node >= 18 (IMP-148). Zero dependência externa. Determinístico: única variação
// entre execuções é meta.gerado_em (as fontes são registradas nos meta.fontes COMO
// PASSADAS na linha de comando — caminhos relativos permanecem relativos).
//
// Bugs do template HTML já corrigidos AQUI (não re-introduzir — sessão 20/11/2026):
//   1. `sel` guarda SEMPRE o wrapper do simulador (normalizar nó cru via pos.get em seleciona(); ativo() também normaliza)
//   2. Repulsão com teto: d2 mínimo 1, rep = Math.min(2600/d2, 40), clamp de velocidade ±40
//   3. Alpha com hard-stop: if (alpha < 0.015){ alpha = 0; return; }
//   4. Pan por DELTA: alvo = {x: view.x + (W/2-sx), y: view.y + (H/2-sy)}
//   5. Clique vs drag: threshold de 4px no mouseup (clique curto seleciona; arrasto move)
//   6. Smoke test não depende de servidor HTTP (o HTML é artefato estático)
//   7. SEM template literals `${}` aninhados no JS embutido do template HTML (colisão quebra o parse do .mjs) — concatenação no JS interno
import { readFileSync, writeFileSync, readdirSync, existsSync, mkdirSync, statSync } from 'node:fs';
import { join, resolve, dirname, basename, extname } from 'node:path';
import { homedir } from 'node:os';
import { fileURLToPath } from 'node:url';

// ---------- Node >= 18 (IMP-148) ----------
const major = Number(process.versions.node.split('.')[0]);
if (!(major >= 18)) {
  console.error('ERRO: Node >= 18 é obrigatório (IMP-148); encontrado ' + process.versions.node + '. Não improvise parse manual em escala — instale Node e re-execute.');
  process.exit(1);
}

// ---------- args ----------
const args = process.argv.slice(2);
function argOf(flag, def) { const i = args.indexOf(flag); return i >= 0 && i + 1 < args.length ? args[i + 1] : def; }
function hasFlag(flag) { return args.includes(flag); }

const CFG_ARG = argOf('--config-root', null);
const PACKS_ARG = argOf('--packs-root', null);
const PACK_ARG = argOf('--pack', null);
const PROJECTS_ARG = argOf('--projects-root', null);
const OUT_ARG = argOf('--out', null);
const SCOPE_ARG = argOf('--scope', null);
const MAXNODES = Math.max(1, Number(argOf('--max-nodes', '3500')) || 3500);
const FROMJSON_ARG = argOf('--from-json', null);

const CFG = resolve(CFG_ARG || join(homedir(), '.config', 'opencode'));
const PACKS = resolve(PACKS_ARG || CFG);
const OUT = resolve(OUT_ARG || join(process.cwd(), 'knowledge-graph'));

const avisosRaw = [];
const aviso = (m) => { if (!avisosRaw.includes(m)) avisosRaw.push(m); };

// caminho EXIBIDO em attrs/meta = como foi passado (determinismo entre máquinas);
// leitura de disco usa resolve().
const packsRootDisp = PACKS_ARG || (CFG_ARG || CFG);
const projectsRootDisp = PROJECTS_ARG || null;

// ---------- helpers ----------
function readText(abs, disp, { obrigatorio = false } = {}) {
  if (!existsSync(abs) || !statSync(abs).isFile()) {
    const m = 'fonte ausente: ' + (disp || abs);
    if (obrigatorio) { console.error('ERRO: ' + m); process.exit(1); }
    aviso(m); return null;
  }
  return readFileSync(abs, 'utf8');
}
const stripBold = (s) => s.replace(/\*\*/g, '').trim();
const arr = (s) => (s && s.startsWith('[')) ? s.slice(1, s.lastIndexOf(']')).split(',').map(x => x.trim()).filter(Boolean) : [];
const hoje = new Date().toISOString().slice(0, 10);

// =====================================================================
// Modo --from-json: regeneração dos 3 artefatos a partir de um grafo
// congelado (sem re-parseio — para reaplicar correção de template).
// =====================================================================
if (FROMJSON_ARG) {
  const srcPath = resolve(FROMJSON_ARG);
  let frozen;
  try {
    frozen = JSON.parse(readFileSync(srcPath, 'utf8'));
  } catch (e) {
    console.error('ERRO: graph.json ilegível em ' + srcPath + ': ' + e.message);
    process.exit(1);
  }
  if (!frozen || !Array.isArray(frozen.nodes) || !Array.isArray(frozen.edges)) {
    console.error('ERRO: ' + srcPath + ' não é um graph.json válido ({meta, nodes, edges}).');
    process.exit(1);
  }
  const meta = frozen.meta && typeof frozen.meta === 'object' ? frozen.meta : {};
  const graph = {
    meta: {
      modo: 'regeneração from-json (grafo congelado)',
      escopo: meta.escopo || 'herdado do grafo congelado',
      fontes: Array.isArray(meta.fontes) ? meta.fontes : [],
      gerado_em: new Date().toISOString(),
      avisos: Array.isArray(meta.avisos) ? meta.avisos : ['regenerado de ' + FROMJSON_ARG + ' — sem re-parseio'],
      podado: meta.podado === undefined ? false : meta.podado,
    },
    nodes: frozen.nodes,
    edges: frozen.edges,
  };
  emitir(graph);
  process.exit(0); // regeneração concluída — não segue para o parse (grafo congelado)
}

// =====================================================================
// Modo parse (global ou delta --scope)
// =====================================================================
const nodes = [], edges = [], nodeById = new Map(), edgeKeys = new Set();
function node(id, type, label, store, attrs = {}) {
  if (nodeById.has(id)) { Object.assign(nodeById.get(id).attrs, attrs); return nodeById.get(id); }
  const n = { id, type, label, store, attrs }; nodeById.set(id, n); nodes.push(n); return n;
}
function edge(from, to, type, attrs = {}) {
  const k = from + '|' + to + '|' + type;
  if (edgeKeys.has(k)) return;
  edgeKeys.add(k);
  edges.push({ from, to, type, attrs });
}

// ---------- descoberta do pack (neutro: sem default de org) ----------
// Explicito --pack > único pack com domains/ > primeiro pack alfabético com domains/ (com aviso).
function discoverPack() {
  if (PACK_ARG) return PACK_ARG;
  const packsDir = join(PACKS, 'packs');
  if (!existsSync(packsDir)) return null;
  const comDomains = readdirSync(packsDir)
    .filter(d => existsSync(join(packsDir, d, 'domains', 'INDEX.md')))
    .sort();
  if (comDomains.length === 0) return null;
  if (comDomains.length > 1) aviso('múltiplos packs com domains/ — usando "' + comDomains[0] + '" (use --pack para fixar)');
  return comDomains[0];
}
const PACK = discoverPack();

// ---------- T5: registry.yaml (STRICT — ausente/inválido ABORTA) ----------
const regDisp = (CFG_ARG || CFG) + '/registry.yaml';
const regTxt = readText(join(CFG, 'registry.yaml'), regDisp, { obrigatorio: true });

const nCfg = node('t5:config:agents', 't5_config', 'AGENTS.md (config)', 'T5', { papel: 'índice lazy + convenções' });
const secRe = /^([a-z_]+):\s*$/;
// parser block: passada única determinística (registry é regular)
function parseBlocks(txt) {
  const out = []; let sec = null, block = null;
  for (const raw of txt.split(/\r?\n/)) {
    const line = raw.replace(/\t/g, '  ');
    const secM = line.match(secRe);
    if (secM) { if (block) { out.push({ sec, block }); block = null; } sec = secM[1]; continue; }
    if (/^ {2}- /.test(line)) {
      if (block) { out.push({ sec, block }); block = null; }
      const t = line.trim();
      if (t.startsWith('- {')) continue; // skill inline — tratado pelo scan dedicado
      block = { id: t.replace(/^- id:\s*/, '').trim(), f: {} };
      continue;
    }
    if (block && /^ {4}[a-z_]+:/.test(line)) {
      const m = line.trim().match(/^([a-z_]+):\s*(.*)$/);
      block.f[m[1]] = m[2].trim();
    }
  }
  if (block) out.push({ sec, block });
  return out.filter(b => b.sec && b.block.id);
}
const blocks = parseBlocks(regTxt);

// skills inline (linhas `  - { id: ..., domain: ..., description: ... }`)
const regSkills = new Map();
for (const raw of regTxt.split(/\r?\n/)) {
  const t = raw.trim();
  if (!t.startsWith('- {') || !/id:\s*/.test(t)) continue;
  if (t.includes('domain:')) {
    const id = (t.match(/id:\s*([^,}]+)/) || [])[1]?.trim();
    const domain = (t.match(/domain:\s*([^,}]+)/) || [])[1]?.trim();
    const desc = (t.match(/description:\s*(.*)\s*\}$/) || t.match(/description:\s*(.*)$/) || [])[1]?.trim();
    if (id) { regSkills.set(id, { domain, description: desc || '' }); node('t5:skill:' + id, 't5_skill', id, 'T5', { domain, description: desc || '' }); }
  }
}
// STRICT: registry sem NENHUM componente reconhecível = inválido → ABORTA
if (blocks.length === 0 && regSkills.size === 0) {
  console.error('ERRO: registry.yaml inválido (nenhum primary/subagent/domain/skill reconhecido): ' + regDisp + ' — camada T5 é a espinha dorsal do grafo (strict).');
  process.exit(1);
}

// nós T5 a partir dos blocos
for (const { sec, block } of blocks) {
  if (sec === 'primary_agents') {
    node('t5:primary:' + block.id, 't5_primary', block.id, 'T5', { description: block.f.description || '' });
  } else if (sec === 'subagents') {
    node('t5:subagent:' + block.id, 't5_subagent', block.id, 'T5', { description: block.f.description || '', owner: block.f.owner || '' });
  } else if (sec === 'domains') {
    node('t5:dominio:' + block.id, 't5_dominio', block.id, 'T5', { orchestrator: block.f.orchestrator || 'null' });
  }
}

// arestas T5
for (const { sec, block } of blocks) {
  const id = block.id;
  if (sec === 'primary_agents') {
    for (const d of arr(block.f.delegates)) edge('t5:primary:' + id, 't5:subagent:' + d, 'delega');
    for (const s of arr(block.f.invokes_direct)) edge('t5:primary:' + id, 't5:skill:' + s, 'invoca_direta');
  } else if (sec === 'subagents') {
    if (block.f.owner) edge('t5:subagent:' + id, 't5:primary:' + block.f.owner, 'owner');
    for (const a of arr(block.f.available_to)) edge('t5:primary:' + a, 't5:subagent:' + id, 'disponivel_para');
    for (const s of arr(block.f.invokes)) edge('t5:subagent:' + id, 't5:skill:' + s, 'invoca');
  } else if (sec === 'domains') {
    const orch = block.f.orchestrator && block.f.orchestrator !== 'null' ? block.f.orchestrator : null;
    if (orch) edge('t5:subagent:' + orch, 't5:dominio:' + id, 'orquestra');
    for (const s of arr(block.f.skills)) edge('t5:dominio:' + id, 't5:skill:' + s, 'agrega');
  }
}
// skills com domain → agrega implícito do domínio T5 (quando o bloco domains não cobrir)
for (const [id, info] of regSkills) {
  if (info.domain && nodeById.has('t5:dominio:' + info.domain) && !edges.some(e => e.type === 'agrega' && e.from === 't5:dominio:' + info.domain && e.to === 't5:skill:' + id))
    edge('t5:dominio:' + info.domain, 't5:skill:' + id, 'agrega', { implicita: 'via domain do registro da skill' });
}

// ---------- T1: domains INDEX.md ----------
const t1Domains = [];
if (PACK) {
  const t1Disp = packsRootDisp + '/packs/' + PACK + '/domains/INDEX.md';
  const t1Txt = readText(join(PACKS, 'packs', PACK, 'domains', 'INDEX.md'), t1Disp);
  if (t1Txt) {
    node('t1:indice', 't1_indice', 'domains/INDEX.md (' + PACK + ')', 'T1', {});
    const chunks = t1Txt.split(/^### /m).slice(1);
    for (const c of chunks) {
      const name = c.split(/\r?\n/)[0].trim();
      if (!name || name.startsWith('#')) continue;
      const ver = (c.match(/\(v(\d+\.\d+\.\d+)\)/) || [])[1] || null;
      const prox = (c.match(/pr[oó]xima[_ ]revis[aã][oõ]?\**\s*:?\s*(\d{4}-\d{2}-\d{2})/i) || (c.match(/proxima_revisao\**\s*:?\s*(\d{4}-\d{2}-\d{2})/i) || []))[1] || null;
      const stale = prox ? prox < hoje : null;
      node('t1:dominio:' + name, 't1_dominio', name, 'T1', { versao: ver ? 'v' + ver : null, proxima_revisao: prox, stale });
      edge('t1:indice', 't1:dominio:' + name, 'indice');
      t1Domains.push(name);
    }
  }
} else {
  aviso('nenhum pack com domains/ encontrado em ' + packsRootDisp + '/packs — T1 pulado');
}

// ---------- T3: rules + índice lazy do AGENTS.md ----------
const lazyTxt = readText(join(CFG, 'AGENTS.md'), (CFG_ARG || CFG) + '/AGENTS.md', { obrigatorio: false }) || '';
if (PACK) {
  const rulesDir = join(PACKS, 'packs', PACK, 'rules');
  if (existsSync(rulesDir)) {
    for (const f of readdirSync(rulesDir).filter(f => f.endsWith('.md') && !f.endsWith('.template.md'))) {
      const name = basename(f, '.md');
      node('t3:rule:' + name, 't3_rule', name, 'T3', { arquivo: 'packs/' + PACK + '/rules/' + f });
      if (new RegExp('rules/' + f).test(lazyTxt)) edge('t5:config:agents', 't3:rule:' + name, 'indice_lazy');
      else aviso('rule fora do índice lazy: ' + f + ' (sinal válido — reportar ao gm-rule-keeper)');
    }
    // registry T3 em yaml (ex.: stack-templates.yaml)
    const yamlRule = join(rulesDir, 'stack-templates.yaml');
    if (existsSync(yamlRule)) node('t3:rule:stack-templates.yaml', 't3_rule', 'stack-templates.yaml', 'T3', { arquivo: 'packs/' + PACK + '/rules/stack-templates.yaml', tipo: 'registry T3 (yaml)' });
  }
}

// ---------- T2: patterns INDEX.md ----------
const t2Slugs = [];
if (PACK) {
  const patDisp = packsRootDisp + '/packs/' + PACK + '/accumulation/patterns/INDEX.md';
  const patTxt = readText(join(PACKS, 'packs', PACK, 'accumulation', 'patterns', 'INDEX.md'), patDisp);
  if (patTxt) {
    node('t2:pack:' + PACK, 't2_pack', 'pack ' + PACK, 'T2', {});
    const rowRe = /\| \[([^\]]+)\]\(([^)]+)\) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \|/;
    let inTable = false;
    for (const line of patTxt.split(/\r?\n/)) {
      if (/^\| Padr/.test(line)) { inTable = true; continue; }
      if (!inTable) continue;
      const m = line.match(rowRe);
      if (!m) { if (inTable && !line.startsWith('|')) inTable = false; continue; }
      const [, slug, , tipo, , projetos, dataRev, status] = m;
      node('t2:pattern:' + slug, 't2_pattern', slug, 'T2', { tipo: tipo.trim(), projetos: projetos.trim(), data_ultima_revisao: dataRev.trim(), status: status.trim() });
      edge('t2:pattern:' + slug, 't2:pack:' + PACK, 'pertence');
      t2Slugs.push(slug);
    }
  }
}

// ---------- fundado_em (skills/agents que citam KNOWLEDGE.md de domínio) ----------
const fundRe = /(?:domains|knowledge)\/([a-z0-9-]+)\/KNOWLEDGE\.md/g;
function scanFund(dir, kind) {
  if (!existsSync(dir)) { aviso('dir ausente para fundado_em: ' + dir); return; }
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const file = kind === 'skill' ? join(p, 'SKILL.md') : p;
    if (!existsSync(file) || !statSync(file).isFile()) continue;
    if (kind === 'skill' && basename(entry) !== entry) continue;
    const txt = readFileSync(file, 'utf8');
    fundRe.lastIndex = 0;
    let m; const found = new Set();
    while ((m = fundRe.exec(txt))) found.add(m[1]);
    for (const d of found) {
      const t1id = 't1:dominio:' + d;
      const from = kind === 'skill' ? 't5:skill:' + entry.replace(/\.md$/, '') : 't5:primary:' + basename(entry, '.md');
      if (nodeById.has(t1id) && nodeById.has(from)) edge(from, t1id, 'fundado_em');
      else if (!nodeById.has(t1id)) aviso('fundado_em: domínio T1 inexistente "' + d + '" citado em ' + file);
    }
  }
}
scanFund(join(CFG, 'skills'), 'skill');
scanFund(join(CFG, 'agents'), 'primary');

// =====================================================================
// T4: módulos de produto (opcional — sem --projects-root, pulado com aviso)
// =====================================================================
const t4Modulos = [];
if (!PROJECTS_ARG) {
  aviso('sem --projects-root — camada T4 (módulos) pulada (modo "estrutura do próprio Golmet")');
} else {
  const projRoot = resolve(PROJECTS_ARG);
  if (!existsSync(projRoot)) {
    aviso('--projects-root inexistente: ' + projectsRootDisp + ' — T4 pulado');
  } else {
    let dirs = readdirSync(projRoot, { withFileTypes: true })
      .filter(d => d.isDirectory())
      .map(d => d.name)
      .filter(name => existsSync(join(projRoot, name, 'REQUIREMENTS.md')) || existsSync(join(projRoot, name, 'delivery')) || existsSync(join(projRoot, name, 'specs')))
      .sort();
    if (SCOPE_ARG) {
      if (!dirs.includes(SCOPE_ARG)) {
        aviso('módulo do --scope não encontrado em ' + projectsRootDisp + ': ' + SCOPE_ARG + ' — T4 vazio (delta)');
      }
      dirs = dirs.filter(d => d === SCOPE_ARG);
    }
    for (const mod of dirs) parseT4Module(projRoot, mod);
  }
}

function parseT4Module(projRoot, mod) {
  t4Modulos.push(mod);
  const modDisp = projectsRootDisp + '/' + mod;
  const modDir = join(projRoot, mod);
  const nMod = node('t4:modulo:' + mod, 't4_modulo', mod, 'T4', { raiz: modDisp });

  // ---- REQUIREMENTS.md: RNs/PENDs + promoção PEND→RN (IMP-094) ----
  const reqTxt = readText(join(modDir, 'REQUIREMENTS.md'), modDisp + '/REQUIREMENTS.md', { obrigatorio: false });
  const rnIds = new Set(), pendIds = new Set();
  if (reqTxt) {
    const secRe4 = /^#{2,3}\s((?:RN|PEND)-\d+)\s*[-—:]\s*(.+)\s*$/;
    for (const line of reqTxt.split(/\r?\n/)) {
      const m = line.match(secRe4);
      if (!m) continue;
      const idFull = m[1], titulo = stripBold(m[2]);
      const tipo = idFull.startsWith('RN') ? 'rn' : 'pend';
      if (tipo === 'rn') rnIds.add(idFull); else pendIds.add(idFull);
      node('t4:' + tipo + ':' + mod + ':' + idFull, 't4_' + tipo, idFull + ' — ' + titulo, 'T4', { modulo: mod });
      edge('t4:modulo:' + mod, 't4:' + tipo + ':' + mod + ':' + idFull, tipo === 'rn' ? 'define' : 'abre_questao');
    }
    // promoção explícita (IMP-094): linha citando PEND-x promovida a RN-y
    for (const line of reqTxt.split(/\r?\n/)) {
      const m1 = line.match(/\bPEND-(\d+)\b[^#\n]{0,160}?promovid[oa]\b[^#\n]{0,160}?\bRN-(\d+)\b/i);
      const m2 = line.match(/\bRN-(\d+)\b[^#\n]{0,160}?promot[ao] d[oa] \bPEND-(\d+)\b/i);
      if (m1) edge('t4:pend:' + mod + ':PEND-' + m1[1], 't4:rn:' + mod + ':RN-' + m1[2], 'promove', { imp: 'IMP-094' });
      else if (m2) edge('t4:pend:' + mod + ':PEND-' + m2[2], 't4:rn:' + mod + ':RN-' + m2[1], 'promove', { imp: 'IMP-094' });
    }
  }

  // ---- DECISIONS.md: ADRs/DECs + refs → decide; citação de pattern → aplica_em ----
  const decTxt = readText(join(modDir, 'DECISIONS.md'), modDisp + '/DECISIONS.md', { obrigatorio: false });
  const decSlugs = new Set();
  if (decTxt) {
    const adrRe = /^##\s((?:ADR|DEC)-\d+)\s*[-—:]\s*(.+)\s*$/;
    let curAdr = null;
    const refRe = /\b(RN|PEND)-(\d+)\b/g;
    for (const line of decTxt.split(/\r?\n/)) {
      const am = line.match(adrRe);
      if (am) {
        curAdr = am[1];
        node('t4:adr:' + mod + ':' + curAdr, 't4_adr', curAdr + ' — ' + stripBold(am[2]), 'T4', { modulo: mod });
        edge('t4:adr:' + mod + ':' + curAdr, 't4:modulo:' + mod, 'artefato');
        continue;
      }
      if (!curAdr) continue;
      if (/^#{1,3}\s/.test(line)) { curAdr = null; continue; }
      refRe.lastIndex = 0;
      let rm;
      while ((rm = refRe.exec(line))) {
        const full = rm[1] + '-' + rm[2];
        const target = rm[1] === 'RN' ? 't4:rn:' + mod + ':' + full : 't4:pend:' + mod + ':' + full;
        if (nodeById.has(target)) edge('t4:adr:' + mod + ':' + curAdr, target, 'decide');
      }
      for (const slug of t2Slugs) if (line.includes(slug)) decSlugs.add(slug);
    }
    for (const slug of decSlugs) if (nodeById.has('t2:pattern:' + slug)) edge('t2:pattern:' + slug, 't4:modulo:' + mod, 'aplica_em');
  }

  // ---- specs/*.md: nós t4_spec ----
  const specsDir = join(modDir, 'specs');
  if (existsSync(specsDir)) {
    for (const f of readdirSync(specsDir).filter(f => f.toLowerCase().endsWith('.md')).sort()) {
      const name = basename(f, '.md');
      node('t4:spec:' + mod + ':' + name, 't4_spec', name + '.md', 'T4', { arquivo: modDisp + '/specs/' + f });
      edge('t4:spec:' + mod + ':' + name, 't4:modulo:' + mod, 'artefato');
    }
  }

  // ---- quality/TRACEABILITY.md: matriz RN × downstream → cobre ----
  const trPath1 = join(modDir, 'quality', 'TRACEABILITY.md');
  const trPath2 = join(modDir, 'QUALITY', 'TRACEABILITY.md');
  const trPath = existsSync(trPath1) ? trPath1 : (existsSync(trPath2) ? trPath2 : trPath1);
  const trTxt = readText(trPath, modDisp + '/quality/TRACEABILITY.md', { obrigatorio: false });
  if (trTxt) {
    const lines = trTxt.split(/\r?\n/);
    let cols = null;
    for (const line of lines) {
      if (cols === null) {
        const hm = line.match(/^\|\s*RN\b([^|]*)\|/);
        if (hm && /\bRN\b/.test(line) && (line.match(/\|/g) || []).length >= 3) {
          cols = line.split('|').slice(2, -1).map(c => c.replace(/\*/g, '').replace(/`/g, '').trim()).filter(Boolean);
        }
        continue;
      }
      const row = line.match(/^\|\s*((?:RN|PEND)-\d+)\s*\|(.+)\|\s*$/);
      if (!row) continue;
      const rnFull = row[1];
      const cells = row[2].split('|').map(c => c.trim());
      cols.forEach((col, i) => {
        const cell = (cells[i] || '').toUpperCase();
        if (!cell || cell === '—' || cell === '-') return;
        const status = cell.startsWith('OK') ? 'OK' : (cell.startsWith('FALHA') ? 'FALHA' : null);
        if (!status) return;
        // resolve o spec por nome de coluna (com ou sem .md) — cria nó se o arquivo não existe (grafo parcial)
        const colClean = col.replace(/\.md$/i, '');
        let specId = 't4:spec:' + mod + ':' + colClean;
        if (!nodeById.has(specId)) node(specId, 't4_spec', colClean + '.md', 'T4', { arquivo: modDisp + '/specs/' + colClean + '.md', origem: 'TRACEABILITY (spec ausente)' });
        const rnNodeId = rnFull.startsWith('RN') ? 't4:rn:' + mod + ':' + rnFull : 't4:pend:' + mod + ':' + rnFull;
        if (!nodeById.has(rnNodeId)) node(rnNodeId, rnFull.startsWith('RN') ? 't4_rn' : 't4_pend', rnFull, 'T4', { modulo: mod, origem: 'TRACEABILITY' });
        edge(rnNodeId, specId, 'cobre', { downstream: colClean, status });
      });
    }
  }

  // ---- delivery/**: features com âncora (IMP-092/093; fallback Tasks.md) ----
  const delDir = join(modDir, 'delivery');
  if (existsSync(delDir)) {
    const feaRe = /^((?:RN|PEND)-\d+)\s*[-—]\s*(.+?)\s*[-—]\s*(Feature|Tasks)\.md$/i;
    const statusRe = /^---\r?\n([\s\S]*?)\r?\n---/;
    function walkDelivery(dir) {
      for (const e of readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
        const p = join(dir, e.name);
        if (e.isDirectory()) { walkDelivery(p); continue; }
        const fm = e.name.match(feaRe);
        if (!fm) continue;
        const [, anchor, titulo, tipoArq] = fm;
        const disp = modDisp + '/delivery/' + e.name;
        const legada = anchor.startsWith('PEND');
        const status = (readFileSync(p, 'utf8').match(statusRe) || ['', ''])[1].match(/^status:\s*(.+)\s*$/m);
        node('t4:feature:' + mod + ':' + anchor, 't4_feature', anchor + ' — ' + stripBold(titulo), 'T4', {
          modulo: mod,
          arquivo: disp,
          tipo: tipoArq.toLowerCase(),
          status: status ? status[1].trim() : null,
          ancora_legada: legada ? true : undefined,
        });
        edge('t4:feature:' + mod + ':' + anchor, 't4:modulo:' + mod, 'artefato');
        if (!legada) edge('t4:rn:' + mod + ':' + anchor, 't4:feature:' + mod + ':' + anchor, 'ancora');
        // âncora PEND legada: NENHUMA aresta inventada (promove/ancora — IMP-094;
        // a promoção é papel do gm-requirements e só vira aresta com evidência
        // explícita na REQUIREMENTS, detectada no parser de promoção acima).
      }
    }
    walkDelivery(delDir);
  }
}

// =====================================================================
// Teto de nós (--max-nodes, default 3500): poda determinística —
// primeiro RN/PEND do modo global (ordenados por id), depois menor grau.
// =====================================================================
let podado = false;
if (nodes.length > MAXNODES) {
  const antes = nodes.length;
  const grau = new Map();
  for (const e of edges) { grau.set(e.from, (grau.get(e.from) || 0) + 1); grau.set(e.to, (grau.get(e.to) || 0) + 1); }
  const drop = new Set();
  const rnPend = nodes.filter(n => n.type === 't4_rn' || n.type === 't4_pend').map(n => n.id).sort();
  for (const id of rnPend) { if (nodes.length - drop.size <= MAXNODES) break; drop.add(id); }
  if (nodes.length - drop.size > MAXNODES) {
    const resto = nodes.filter(n => !drop.has(n.id) && n.type !== 't5_config')
      .map(n => ({ id: n.id, g: grau.get(n.id) || 0 }))
      .sort((a, b) => a.g - b.g || a.id.localeCompare(b.id));
    for (const r of resto) { if (nodes.length - drop.size <= MAXNODES) break; drop.add(r.id); }
  }
  for (const id of drop) { nodeById.delete(id); }
  for (let i = nodes.length - 1; i >= 0; i--) if (drop.has(nodes[i].id)) nodes.splice(i, 1);
  podado = { antes, removidos: drop.size, criterio: 'RN/PEND do modo global primeiro, depois menor grau (id asc)' };
}

// ---------- saneamento: arestas com ponta solta ----------
const validas = [];
for (const e of edges) {
  if (nodeById.has(e.from) && nodeById.has(e.to)) validas.push(e);
  else aviso('aresta descartada (ponta solta): ' + e.from + ' --' + e.type + '--> ' + e.to);
}
validas.sort((a, b) => a.type.localeCompare(b.type) || a.from.localeCompare(b.from) || a.to.localeCompare(b.to));
edges.length = 0; edges.push(...validas);
nodes.sort((a, b) => a.store.localeCompare(b.store) || a.type.localeCompare(b.type) || a.id.localeCompare(b.id));

// ---------- meta + montagem integral em memória ----------
const porTipo = {};
for (const n of nodes) porTipo[n.type] = (porTipo[n.type] || 0) + 1;
const staleT1 = nodes.filter(n => n.attrs.stale === true).map(n => n.label);
const fontes = [
  regDisp,
  (CFG_ARG || CFG) + '/AGENTS.md',
];
if (PACK) {
  fontes.push(packsRootDisp + '/packs/' + PACK + '/domains/INDEX.md',
    packsRootDisp + '/packs/' + PACK + '/rules/',
    packsRootDisp + '/packs/' + PACK + '/accumulation/patterns/INDEX.md');
}
fontes.push((CFG_ARG || CFG) + '/skills/*/SKILL.md', (CFG_ARG || CFG) + '/agents/*.md');
if (PROJECTS_ARG && existsSync(resolve(PROJECTS_ARG))) fontes.push(projectsRootDisp + '/{REQUIREMENTS.md, DECISIONS.md, quality/TRACEABILITY.md, delivery/**, specs/}');
if (SCOPE_ARG && t4Modulos.length === 0) aviso('delta sem módulo correspondente — grafo sem T4');
const graph = {
  meta: {
    modo: SCOPE_ARG ? 'delta (scope: ' + SCOPE_ARG + ')' : 'global',
    escopo: PROJECTS_ARG && t4Modulos.length ? (SCOPE_ARG ? SCOPE_ARG : t4Modulos.join(', ')) : 'estrutura do próprio Golmet (T1-T5)' + (PACK ? ' · pack ' + PACK : ''),
    fontes,
    gerado_em: new Date().toISOString(),
    avisos: avisosRaw.slice(),
    podado,
  },
  nodes, edges,
};
emitir(graph);

// =====================================================================
// Emissão — os 3 artefatos (só escreve depois de montar tudo)
// =====================================================================
function emitir(graph) {
  const gJson = JSON.stringify(graph, null, 2);
  const md = renderMd(graph);
  const html = renderHtml(graph);
  mkdirSync(OUT, { recursive: true });
  writeFileSync(join(OUT, 'graph.json'), gJson, 'utf8');
  writeFileSync(join(OUT, 'knowledge-graph.html'), html, 'utf8');
  writeFileSync(join(OUT, 'knowledge-graph.md'), md, 'utf8');

  // relatório no console
  const porTipo2 = {};
  for (const n of graph.nodes) porTipo2[n.type] = (porTipo2[n.type] || 0) + 1;
  const stale2 = graph.nodes.filter(n => n.attrs && n.attrs.stale === true).map(n => n.label);
  console.log('OK: ' + graph.nodes.length + ' nós · ' + graph.edges.length + ' arestas · ' + OUT);
  console.log('Por tipo: ' + JSON.stringify(porTipo2));
  console.log('T1 stale: ' + (stale2.length ? stale2.join(', ') : 'nenhuma'));
  if (graph.meta.podado) console.log('PODADO: ' + JSON.stringify(graph.meta.podado));
  console.log('Avisos: ' + graph.meta.avisos.length);
  for (const a of graph.meta.avisos.slice(0, 15)) console.log('  - ' + a);
}

function renderMd(graph) {
  const grauOf = (id) => { let g = 0; for (const e of graph.edges) if (e.from === id || e.to === id) g++; return g; };
  const top = [...graph.nodes].sort((a, b) => grauOf(b.id) - grauOf(a.id) || a.id.localeCompare(b.id)).slice(0, 30);
  const ids = new Set(top.map(n => n.id));
  const mEdges = graph.edges.filter(e => ids.has(e.from) && ids.has(e.to)).slice(0, 60);
  const sanitize = (s) => s.replace(/[^a-zA-Z0-9]/g, '_');
  const stale = graph.nodes.filter(n => n.attrs && n.attrs.stale === true).map(n => n.label);
  const porTipo3 = {};
  for (const n of graph.nodes) porTipo3[n.type] = (porTipo3[n.type] || 0) + 1;
  return '# Grafo do Conhecimento (T1-T5)\n'
    + '\n'
    + '> Gerado por gm-knowledge-graph (scripts/build-knowledge-graph.mjs — determinístico).\n'
    + '> Gerado em ' + graph.meta.gerado_em + ' · modo: ' + graph.meta.modo + ' · escopo: ' + graph.meta.escopo + '\n'
    + '\n'
    + '## Contagem por tipo\n'
    + '\n'
    + '| Tipo | Nós |\n|---|---|\n'
    + Object.entries(porTipo3).map(([t, c]) => '| ' + t + ' | ' + c + ' |').join('\n')
    + '\n\n'
    + '**Arestas:** ' + graph.edges.length + (graph.meta.podado ? ' · **PODADO:** ' + graph.meta.podado.removidos + ' nós (teto ' + MAXNODES + ')' : '') + '\n'
    + '\n'
    + '**Bases T1 stale:** ' + (stale.length ? stale.join(', ') : 'nenhuma') + '\n'
    + '\n'
    + '## Mermaid (top ' + top.length + ' nós por grau)\n'
    + '\n'
    + '```mermaid\ngraph LR\n'
    + top.map(n => '  ' + sanitize(n.id) + '["' + n.label.replace(/"/g, "'") + '"]').join('\n')
    + '\n'
    + mEdges.map(e => '  ' + sanitize(e.from) + ' -->|' + e.type + '| ' + sanitize(e.to)).join('\n')
    + '\n```\n'
    + '\n'
    + '## Avisos (' + graph.meta.avisos.length + ')\n'
    + '\n'
    + (graph.meta.avisos.length ? graph.meta.avisos.map(a => '- ' + a).join('\n') : '- nenhum')
    + '\n';
}

function renderHtml(graph) {
  const dados = { gerado_em: graph.meta.gerado_em, nodes: graph.nodes, edges: graph.edges };
  const jsonEsc = JSON.stringify(dados).replace(/</g, '\\u003c');
  const html = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>GOLMET — Grafo do Conhecimento (T1-T5)</title>
<style>
:root{--bg:#1A1D21;--bg2:#22262c;--ouro:#E8B44A;--ambar:#FF6B35;--tx:#EDEAE2;--tx2:#9aa0a8;--ln:#3a4048}
*{margin:0;padding:0;box-sizing:border-box}html,body{height:100%;overflow:hidden}
body{background:var(--bg);color:var(--tx);font-family:'Archivo','Segoe UI',system-ui,sans-serif;display:flex;flex-direction:column}
header{display:flex;align-items:center;gap:1rem;padding:.7rem 1.2rem;border-bottom:1px solid var(--ln);flex-wrap:wrap}
header .marca{color:var(--ouro);font-weight:800;letter-spacing:.12em}
header .sub{color:var(--tx2);font-size:.8rem}
#busca{background:#121417;border:1px solid var(--ln);border-radius:6px;color:var(--tx);padding:.4rem .8rem;font-size:.85rem;width:220px;font-family:inherit}
#busca:focus{outline:none;border-color:var(--ouro)}
.filtros{display:flex;gap:.9rem;flex-wrap:wrap;font-size:.8rem;color:var(--tx2)}
.filtros label{cursor:pointer;display:flex;align-items:center;gap:.35rem}
.filtros input{accent-color:var(--ouro)}
#stats{margin-left:auto;color:var(--tx2);font-size:.8rem}
main{flex:1;position:relative}
canvas{position:absolute;inset:0;width:100%;height:100%;cursor:grab}
canvas:active{cursor:grabbing}
#painel{position:absolute;top:1rem;right:1rem;width:320px;max-height:70vh;overflow:auto;background:rgba(18,20,23,.95);border:1px solid var(--ln);border-radius:8px;padding:1rem;display:none;font-size:.85rem}
#painel h3{color:var(--ouro);margin-bottom:.4rem;font-size:1rem}
#painel .tipo{color:var(--ambar);font-size:.72rem;text-transform:uppercase;letter-spacing:.12em;margin-bottom:.6rem}
#painel table{width:100%;border-collapse:collapse}
#painel td{border-top:1px solid var(--ln);padding:.3rem 0;vertical-align:top}
#painel td:first-child{color:var(--tx2);padding-right:.6rem;white-space:nowrap}
#vizinhos{margin-top:.6rem;color:var(--tx2)}
#vizinhos .v{color:var(--tx);cursor:pointer}
#vizinhos .v:hover{color:var(--ouro)}
#reset{position:absolute;bottom:1rem;right:1rem;background:var(--bg2);color:var(--tx);border:1px solid var(--ln);border-radius:6px;padding:.4rem .9rem;cursor:pointer;font-family:inherit}
#reset:hover{border-color:var(--ouro);color:var(--ouro)}
#legenda{position:absolute;bottom:1rem;left:1rem;background:rgba(18,20,23,.9);border:1px solid var(--ln);border-radius:8px;padding:.7rem 1rem;font-size:.75rem;color:var(--tx2)}
#legenda .l{display:flex;align-items:center;gap:.45rem;margin:.15rem 0}
.dot{width:10px;height:10px;border-radius:50%;display:inline-block}
</style></head><body>
<header>
  <span class="marca">GOLMET</span><span class="sub">grafo do conhecimento T1-T5 — leitura da base</span>
  <input id="busca" placeholder="buscar nó…" autocomplete="off">
  <div class="filtros" id="filtros"></div>
  <span id="stats"></span>
</header>
<main><canvas id="cv"></canvas>
<div id="painel"><div class="tipo" id="pTipo"></div><h3 id="pTitulo"></h3><div id="pAttrs"></div><div id="vizinhos"></div></div>
<button id="reset">reset view</button>
<div id="legenda"></div>
</main>
<script id="grafo" type="application/json">__GRAPH_JSON__</script>
<script>
const G = JSON.parse(document.getElementById('grafo').textContent);
const CORES = { T1:'#4da3ff', T2:'#b48cff', T3:'#4dd0a3', T4:'#e05c7a', T5:'#E8B44A' };
const NOMES = { t5_config:'T5 · config', t5_primary:'T5 · primary', t5_subagent:'T5 · subagent', t5_skill:'T5 · skill', t5_dominio:'T5 · domínio', t1_indice:'T1 · índice', t1_dominio:'T1 · domínio', t3_rule:'T3 · rule', t2_pack:'T2 · pack', t2_pattern:'T2 · pattern', t4_modulo:'T4 · módulo', t4_rn:'T4 · RN', t4_pend:'T4 · pendência', t4_adr:'T4 · decisão', t4_feature:'T4 · feature', t4_spec:'T4 · spec' };
const cv = document.getElementById('cv'), ctx = cv.getContext('2d');
const painel = document.getElementById('painel'), stats = document.getElementById('stats');
let W=0,H=0,DPR=window.devicePixelRatio||1;
function resize(){ W=cv.clientWidth; H=cv.clientHeight; cv.width=W*DPR; cv.height=H*DPR; ctx.setTransform(DPR,0,0,DPR,0,0); }
window.addEventListener('resize', resize);

// estado do view
let view = { x:0, y:0, k:1 }, sel = null, hover = null, drag = null, panning = false, moved = false;

// física force-directed (O(n²)) — bugs corrigidos: teto de repulsão, clamp de
// velocidade e hard-stop de alpha (sem isso a simulação diverge / nunca assenta)
const N = G.nodes.map(function(n){ return { n: n, x:(Math.random()-.5)*800, y:(Math.random()-.5)*800, vx:0, vy:0 }; });
const pos = new Map(N.map(function(d){ return [d.n.id, d]; }));
const grau = new Map();
for (const e of G.edges){ grau.set(e.from,(grau.get(e.from)||0)+1); grau.set(e.to,(grau.get(e.to)||0)+1); }
let alpha = 1, layouting = false;
function tick(){
  if (alpha <= 0) return;
  if (alpha < 0.015){ alpha = 0; return; } // hard-stop — congela (drag reenergiza)
  for (let i=0;i<N.length;i++) for (let j=i+1;j<N.length;j++){
    const a=N[i], b=N[j];
    let dx=b.x-a.x, dy=b.y-a.y;
    let d2=dx*dx+dy*dy; if (d2<1) d2=1;
    const d=Math.sqrt(d2);
    const rep = Math.min(2600/d2, 40); let fx=dx/d*rep, fy=dy/d*rep;
    a.vx-=fx; a.vy-=fy; b.vx+=fx; b.vy+=fy;
  }
  for (const e of G.edges){
    const a=pos.get(e.from), b=pos.get(e.to); if(!a||!b) continue;
    let dx=b.x-a.x, dy=b.y-a.y, d=Math.sqrt(dx*dx+dy*dy)||1;
    const alvo=95, f=(d-alvo)/d*0.06;
    a.vx+=dx*f; a.vy+=dy*f; b.vx-=dx*f; b.vy-=dy*f;
  }
  for (const d of N){
    if (d===drag) continue;
    d.vx*=0.85; d.vy*=0.85;
    d.vx=Math.max(-40,Math.min(40,d.vx)); d.vy=Math.max(-40,Math.min(40,d.vy));
    d.x+=d.vx*alpha*2; d.y+=d.vy*alpha*2;
  }
  alpha*=0.995;
}
// auto-fit: bbox + margem 40px por lado (contrato SKILL.md)
function autoFit(){
  if (!N.length) return;
  let x0=1e9,y0=1e9,x1=-1e9,y1=-1e9;
  for (const d of N){ x0=Math.min(x0,d.x); y0=Math.min(y0,d.y); x1=Math.max(x1,d.x); y1=Math.max(y1,d.y); }
  const mx=(x1-x0)||1, my=(y1-y0)||1;
  const k=Math.min((W-80)/mx,(H-80)/my,2.2);
  const cx=(x0+x1)/2, cy=(y0+y1)/2;
  view.k=k; view.x=W/2-cx*k; view.y=H/2-cy*k;
}
function toScreen(d){ return [d.x*view.k+view.x, d.y*view.k+view.y]; }
function toWorld(px,py){ return [(px-view.x)/view.k, (py-view.y)/view.k]; }
function ativo(nRaw){
  const w = nRaw.n ? nRaw : pos.get(nRaw.id); // normaliza wrapper/raw — inconsistência apagava o canvas
  if (!w) return false;
  if (w.n._off) return false;
  if (!sel) return true;
  return w===sel || G.edges.some(function(e){ return (e.from===sel.n.id&&e.to===w.n.id)||(e.to===sel.n.id&&e.from===w.n.id); });
}
function raio(n){ return 5+Math.min(9,(grau.get(n.id)||0)*0.6); }
function draw(){
  ctx.clearRect(0,0,W,H);
  ctx.save();
  ctx.translate(view.x,view.y); ctx.scale(view.k,view.k);
  for (const e of G.edges){
    const a=pos.get(e.from), b=pos.get(e.to); if(!a||!b) continue;
    const ligada = sel && (a===sel || b===sel);
    const on = !sel || a===sel || b===sel;
    ctx.strokeStyle = !sel ? 'rgba(154,160,168,0.28)' : (ligada ? 'rgba(232,180,74,0.9)' : (on ? 'rgba(154,160,168,0.6)' : 'rgba(154,160,168,0.07)'));
    ctx.lineWidth = ligada ? 2.2 : 1;
    ctx.beginPath(); ctx.moveTo(a.x,a.y); ctx.lineTo(b.x,b.y); ctx.stroke();
  }
  if (sel){
    ctx.font = '600 10px Archivo, sans-serif';
    ctx.textAlign = 'center';
    for (const e of G.edges){
      if (e.from!==sel.n.id && e.to!==sel.n.id) continue;
      const a=pos.get(e.from), b=pos.get(e.to); if(!a||!b) continue;
      const mx=(a.x+b.x)/2, my=(a.y+b.y)/2;
      ctx.lineWidth = 3; ctx.strokeStyle = 'rgba(26,29,33,0.85)';
      ctx.strokeText(e.type, mx, my-4);
      ctx.fillStyle = '#FF6B35';
      ctx.fillText(e.type, mx, my-4);
    }
    ctx.textAlign = 'start';
  }
  for (const d of N){
    const on = ativo(d.n);
    ctx.globalAlpha = on?1:0.12;
    ctx.beginPath(); ctx.arc(d.x,d.y,raio(d.n),0,7);
    ctx.fillStyle = d===sel ? '#FF6B35' : (CORES[d.n.store]||'#999');
    ctx.fill();
    if (d===sel || (sel && on) || (N.length<220 && view.k>0.7)){
      ctx.fillStyle = on?'#EDEAE2':'rgba(237,234,226,0.25)';
      ctx.font = (d===sel?'600 ':'')+'10px Archivo, sans-serif';
      ctx.fillText(d.n.label, d.x+raio(d.n)+3, d.y+3);
    }
    ctx.globalAlpha = 1;
  }
  ctx.restore();
}
// layout assíncrono em chunks (>= 200 nós; < 200 síncrono) com progresso no #stats;
// pan/zoom/seleção funcionam durante o layout (event-driven, um tick por frame)
function statsFinal(){ stats.textContent = N.length + ' nós · ' + G.edges.length + ' arestas · ' + G.gerado_em.slice(0,10); }
function frame(){
  if (layouting){
    tick();
    if (alpha <= 0 || alpha < 0.015){ alpha = 0; layouting = false; autoFit(); statsFinal(); }
    else stats.textContent = 'layout ' + Math.max(0,Math.round((1-alpha)*100)) + '% · ' + N.length + ' nós · ' + G.edges.length + ' arestas';
  }
  draw();
  requestAnimationFrame(frame);
}
function reenergiza(){ if (!layouting){ layouting = true; } }

// interação (bugs corrigidos: pan por delta, clique vs drag com threshold 4px)
let lastMX=0,lastMY=0,downX=0,downY=0;
cv.addEventListener('mousedown', function(e){ moved=false; downX=e.offsetX; downY=e.offsetY; const w=toWorld(e.offsetX,e.offsetY); const d=pega(w[0],w[1]); if(d){ drag=d; } else { panning=true; } lastMX=e.offsetX; lastMY=e.offsetY; });
cv.addEventListener('mousemove', function(e){
  const dist = Math.abs(e.offsetX-downX)+Math.abs(e.offsetY-downY);
  if (drag){ drag.x+= (e.offsetX-lastMX)/view.k; drag.y+=(e.offsetY-lastMY)/view.k; moved = dist > 4; if (moved){ alpha=Math.max(alpha,0.3); reenergiza(); } }
  else if (panning){ view.x+=e.offsetX-lastMX; view.y+=e.offsetY-lastMY; moved = dist > 4; }
  lastMX=e.offsetX; lastMY=e.offsetY;
});
window.addEventListener('mouseup', function(e){
  if (drag){
    const clique = Math.abs(e.offsetX-downX)+Math.abs(e.offsetY-downY) <= 4;
    if (clique) seleciona(drag); // clique curto num nó = selecionar; arrasto = mover
    drag=null;
  }
  else if (panning && !moved){ const w=toWorld(e.offsetX,e.offsetY); seleciona(pega(w[0],w[1]) || null); }
  panning=false;
});
cv.addEventListener('wheel', function(e){ e.preventDefault(); const f=e.deltaY<0?1.12:0.89; const w=toWorld(e.offsetX,e.offsetY); view.k=Math.max(0.15,Math.min(4,view.k*f)); view.x=e.offsetX-w[0]*view.k; view.y=e.offsetY-w[1]*view.k; },{passive:false});
function pega(wx,wy){ let best=null,bd=14/view.k; for(const d of N){ const dx=d.x-wx,dy=d.y-wy,dd=Math.hypot(dx,dy); if(dd<Math.max(bd,raio(d.n))){best=d;bd=dd;} } return best; }

function seleciona(nRaw){
  // aceita wrapper (do simulador) ou nó cru (do JSON) — SEMPRE normaliza p/ wrapper
  let n = null;
  if (nRaw) n = nRaw.n ? nRaw : (pos.get(nRaw.id) || null);
  sel = n; painel.style.display=n?'block':'none';
  if(!n) return;
  document.getElementById('pTipo').textContent = NOMES[n.n.type]||n.n.type;
  document.getElementById('pTitulo').textContent = n.n.label;
  const at = Object.entries(n.n.attrs).filter(function(kv){ return kv[1]!==null&&kv[1]!==undefined&&kv[1]!==''; });
  document.getElementById('pAttrs').innerHTML = at.length ? '<table>'+at.map(function(kv){ var k=kv[0], v=String(kv[1]); if(v.length>140) v=v.slice(0,140)+'…'; return '<tr><td>'+k+'</td><td>'+v+'</td></tr>'; }).join('')+'</table>' : '';
  const viz = G.edges.filter(function(e){ return e.from===n.n.id||e.to===n.n.id; }).map(function(e){
    const outro = e.from===n.n.id?e.to:e.from; const dn=pos.get(outro);
    return '<span class="v" data-id="'+outro+'">'+(dn?dn.n.label:outro)+'</span> <span style="color:#555">('+e.type+')</span>';
  }).join(' · ');
  document.getElementById('vizinhos').innerHTML = viz? '<strong style="color:#E8B44A">Vizinhos</strong><br>'+viz : '';
  painel.querySelectorAll('.v').forEach(function(el){ el.addEventListener('click',function(){ const d=pos.get(el.dataset.id); if(d){ seleciona(d); const s=toScreen(d); // pan por DELTA — computar absoluto jogava o nó no canto
    animaPan({ x: view.x + (W/2-s[0]), y: view.y + (H/2-s[1]) }); } }); });
}
function animaPan(alvo){ const i0={x:view.x,y:view.y}; let t=0; (function passo(){ t=Math.min(1,t+0.12); view.x=i0.x+(alvo.x-i0.x)*(1-Math.pow(1-t,3)); view.y=i0.y+(alvo.y-i0.y)*(1-Math.pow(1-t,3)); if(t<1) requestAnimationFrame(passo); })(); }

// filtros por store
const stores = ['T1','T2','T3','T4','T5'];
const fdiv = document.getElementById('filtros');
for (const s of stores){
  const lb=document.createElement('label'); lb.innerHTML='<input type="checkbox" checked data-store="'+s+'"> '+s+' <span style="color:#6b7280">'+G.nodes.filter(function(n){return n.store===s}).length+'</span>';
  lb.querySelector('input').addEventListener('change', function(ev){
    const on=ev.target.checked;
    for (const d of N) if (d.n.store===s) d.n._off=!on;
  });
  fdiv.appendChild(lb);
}
// busca
document.getElementById('busca').addEventListener('input', function(ev){
  const q=ev.target.value.toLowerCase().trim();
  const hit = q ? G.nodes.find(function(n){ return n.label.toLowerCase().includes(q); }) : null;
  if (hit){ const d=pos.get(hit.id); seleciona(hit); const s=toScreen(d); animaPan({ x: view.x + (W/2-s[0]), y: view.y + (H/2-s[1]) }); }
  else seleciona(null);
});
document.getElementById('reset').addEventListener('click', function(){ autoFit(); });

// legenda
const leg=document.getElementById('legenda');
leg.innerHTML = stores.map(function(s){ return '<div class="l"><span class="dot" style="background:'+CORES[s]+'"></span>'+s+' · '+G.nodes.filter(function(n){return n.store===s}).length+' nós</div>'; }).join('') + '<div class="l" style="margin-top:.4rem">'+G.edges.length+' arestas</div>';

resize(); autoFit(); statsFinal();
if (N.length >= 200) { layouting = true; } // assíncrono com indicador no #stats
else { var guard=0; while (alpha >= 0.015 && guard < 4000){ tick(); guard++; } alpha = 0; autoFit(); statsFinal(); }
frame();
</script></body></html>`;
  // replacement via função: labels podem conter "$" (ex.: R$) — com string de
  // substituição direta, $&/$' do JSON corromperiam o HTML
  return html.replace('__GRAPH_JSON__', () => jsonEsc);
}
