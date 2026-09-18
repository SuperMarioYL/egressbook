#!/usr/bin/env python3
"""egressbook web asset generator — reads web/palette.json semantic tokens and
emits every themed SVG the site references (scene fallbacks, story pairs,
demo step panels). Re-run together with scripts/palette.js on any color change."""
import json, os, sys

WEB = os.path.join(os.path.dirname(__file__), "..", "web")
PAL = json.load(open(os.path.join(WEB, "palette.json")))
PID = PAL["id"]
D, L = PAL["dark"], PAL["light"]

def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

def svg(path, w, h, body, theme):
    t = D if theme == "dark" else L
    root = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" '
            f'viewBox="0 0 {w} {h}" data-palette="{PID}" role="img">')
    bg = f'<rect width="{w}" height="{h}" rx="10" fill="{t["bg"]}"/>'
    out = os.path.join(WEB, "assets", path)
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w") as f:
        f.write(root + bg + body(t) + "</svg>\n")
    print("wrote", path)

MONO = "ui-monospace, Menlo, monospace"

def term_panel(t, lines, w=420, h=150, title="egressbook"):
    parts = [
      f'<rect x="8" y="8" width="{w-16}" height="{h-16}" rx="8" fill="{t["panel"]}" stroke="{t["line"]}"/>',
      f'<text x="20" y="32" font-family="{MONO}" font-size="12" fill="{t["muted"]}">{esc(title)}</text>']
    y = 54
    for kind, text in lines:
        color = {"cmd": t["primary"], "ok": t["secondary"], "warn": t["highlight"],
                 "dim": t["muted"], "out": t["ink"]}.get(kind, t["ink"])
        parts.append(f'<text x="20" y="{y}" font-family="{MONO}" font-size="12" fill="{color}">{esc(text)}</text>')
        y += 20
    return "".join(parts)

# --- demo step panels (single-path, dark styling matches the recorded run) ---
svg("scan-dark.svg", 420, 190, lambda t: term_panel(t, [
    ("cmd", "$ egressbook scan"),
    ("out", "zcode        sessions: found=1 parsed=1"),
    ("out", "codex        sessions: found=1 parsed=1"),
    ("out", "claude-code  sessions: found=1 parsed=1"),
    ("warn", "gap: system prompt not written to session logs"),
    ("warn", "gap: model provider not recorded per-session"),
], h=190), "dark")

svg("ledger-dark.svg", 420, 190, lambda t: term_panel(t, [
    ("cmd", "$ egressbook ledger"),
    ("out", "DESTINATION MAP (outbound bytes)"),
    ("ok", "  Zhipu (GLM)        240 bytes"),
    ("ok", "  OpenAI             154 bytes"),
    ("ok", "  Anthropic (Claude)  63 bytes"),
    ("dim", "signature: ed25519  sha256=6156ee55…"),
], h=190), "dark")

svg("diff-dark.svg", 420, 170, lambda t: term_panel(t, [
    ("cmd", "$ egressbook diff --since 1w"),
    ("out", "NEW DESTINATIONS"),
    ("ok", "  + Zhipu (GLM) (+240 bytes)"),
    ("out", "NEW PAYLOAD KINDS"),
    ("warn", "  + git_artifact"),
], h=170), "dark")

svg("verify-dark.svg", 420, 120, lambda t: term_panel(t, [
    ("cmd", "$ egressbook verify ledger.json"),
    ("ok", "signature OK (ed25519, 11 items)"),
], h=120), "dark")

# --- scene fallbacks: machine node wired to provider nodes ---
def scene(t):
    p = []
    cx, cy = 110, 150
    p.append(f'<circle cx="{cx}" cy="{cy}" r="34" fill="{t["panel"]}" stroke="{t["primary"]}" stroke-width="2"/>')
    p.append(f'<text x="{cx}" y="{cy+4}" text-anchor="middle" font-family="{MONO}" font-size="11" fill="{t["ink"]}">your</text>')
    p.append(f'<text x="{cx}" y="{cy+17}" text-anchor="middle" font-family="{MONO}" font-size="11" fill="{t["ink"]}">machine</text>')
    providers = [("Zhipu (GLM)", "240 B", 330, 50),
                 ("OpenAI", "154 B", 355, 140),
                 ("Anthropic", "63 B", 330, 230)]
    for name, b, x, y in providers:
        p.append(f'<line x1="{cx+34}" y1="{cy}" x2="{x-52}" y2="{y}" stroke="{t["secondary"]}" stroke-width="1.6" opacity="0.85"/>')
        p.append(f'<rect x="{x-52}" y="{y-22}" width="124" height="44" rx="8" fill="{t["panel"]}" stroke="{t["line"]}"/>')
        p.append(f'<text x="{x+10}" y="{y-4}" text-anchor="middle" font-family="{MONO}" font-size="11" fill="{t["ink"]}">{esc(name)}</text>')
        p.append(f'<text x="{x+10}" y="{y+13}" text-anchor="middle" font-family="{MONO}" font-size="11" fill="{t["primary"]}">{b}</text>')
    return "".join(p)

svg("scene-dark.svg", 460, 280, scene, "dark")
svg("scene-light.svg", 460, 280, scene, "light")

# --- story: architecture (parse → classify → sign) ---
def arch(t):
    p = []
    stages = [("parse", "~/.zcode  ~/.codex  ~/.claude"),
              ("classify", "6 payload kinds"),
              ("sign", "ed25519 ledger")]
    x = 24
    for i, (name, sub) in enumerate(stages):
        p.append(f'<rect x="{x}" y="70" width="120" height="70" rx="10" fill="{t["panel"]}" stroke="{t["primary"]}" stroke-width="1.6"/>')
        p.append(f'<text x="{x+60}" y="100" text-anchor="middle" font-family="{MONO}" font-size="14" fill="{t["primary"]}">{name}</text>')
        p.append(f'<text x="{x+60}" y="120" text-anchor="middle" font-family="{MONO}" font-size="10" fill="{t["muted"]}">{esc(sub)}</text>')
        if i < 2:
            p.append(f'<line x1="{x+120}" y1="105" x2="{x+152}" y2="105" stroke="{t["secondary"]}" stroke-width="2" marker-end="url(#a{i})"/>')
        x += 152
    defs = "".join(f'<marker id="a{i}" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L6,3 L0,6 z" fill="{t["secondary"]}"/></marker>' for i in range(2))
    return defs + "".join(p)

svg("architecture-dark.svg", 500, 200, arch, "dark")
svg("architecture-light.svg", 500, 200, arch, "light")

# --- story: destination map bars ---
def destmap(t):
    rows = [("Zhipu (GLM)", 240), ("OpenAI", 154), ("Anthropic (Claude)", 63)]
    p, y = [], 50
    maxv = max(v for _, v in rows)
    for name, v in rows:
        w = int(220 * v / maxv) + 4
        p.append(f'<text x="24" y="{y}" font-family="{MONO}" font-size="12" fill="{t["ink"]}">{esc(name)}</text>')
        p.append(f'<rect x="200" y="{y-14}" width="{w}" height="18" rx="4" fill="{t["primary"]}" opacity="0.9"/>')
        p.append(f'<text x="{204+w}" y="{y}" font-family="{MONO}" font-size="12" fill="{t["muted"]}">{v} B</text>')
        y += 44
    return "".join(p)

svg("destmap-dark.svg", 480, 200, destmap, "dark")
svg("destmap-light.svg", 480, 200, destmap, "light")

# --- story: delta between two ledgers ---
def delta(t):
    p = [
      f'<rect x="30" y="60" width="150" height="90" rx="10" fill="{t["panel"]}" stroke="{t["line"]}"/>',
      f'<text x="105" y="95" text-anchor="middle" font-family="{MONO}" font-size="12" fill="{t["muted"]}">ledger · 2w ago</text>',
      f'<text x="105" y="118" text-anchor="middle" font-family="{MONO}" font-size="12" fill="{t["muted"]}">2 destinations</text>',
      f'<rect x="320" y="60" width="150" height="90" rx="10" fill="{t["panel"]}" stroke="{t["primary"]}" stroke-width="1.6"/>',
      f'<text x="395" y="95" text-anchor="middle" font-family="{MONO}" font-size="12" fill="{t["ink"]}">ledger · now</text>',
      f'<text x="395" y="118" text-anchor="middle" font-family="{MONO}" font-size="12" fill="{t["primary"]}">3 destinations</text>',
      '<defs><marker id="da" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">'
      f'<path d="M0,0 L6,3 L0,6 z" fill="{t["secondary"]}"/></marker></defs>',
      f'<line x1="185" y1="105" x2="312" y2="105" stroke="{t["secondary"]}" stroke-width="2" marker-end="url(#da)"/>',
      f'<text x="250" y="90" text-anchor="middle" font-family="{MONO}" font-size="11" fill="{t["highlight"]}">+ Zhipu (GLM)</text>',
    ]
    return "".join(p)

svg("delta-dark.svg", 500, 200, delta, "dark")
svg("delta-light.svg", 500, 200, delta, "light")

print(f"done — all SVGs carry data-palette={PID}")
