# Generates Axon's original themed-space artwork (EP-38) as SVG:
#   python3 scripts/spaces.py static/spaces
# Every drawing is made here from geometry; nothing is copied from elsewhere,
# so the artwork is the project's own and ships under its licence. Each space
# has a light and a dark variant.
import math, random, sys, os
out = sys.argv[1]
W, H = 1600, 1000
random.seed(7)

def svg(body, defs=''):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" preserveAspectRatio="xMidYMid slice">'
            f'<defs>{defs}</defs>{body}</svg>\n')

def grain(id_, freq, opacity, color):
    return (f'<filter id="{id_}" x="0" y="0" width="100%" height="100%"><feTurbulence type="fractalNoise" baseFrequency="{freq}" numOctaves="3" seed="4"/>'
            f'<feColorMatrix values="0 0 0 0 {color[0]} 0 0 0 0 {color[1]} 0 0 0 0 {color[2]} 0 0 0 {opacity} 0"/></filter>')

def f(x): return f'{x:.1f}'

# --- Da Vinci notebook: parchment, sepia sketches, mirror script ---
def notebook(ink, paper_hi, paper_lo, alpha):
    d = grain('g', 0.8, 0.10, (0.35, 0.22, 0.08)) + (
        f'<radialGradient id="p" cx="50%" cy="45%" r="75%"><stop offset="0" stop-color="{paper_hi}"/><stop offset="1" stop-color="{paper_lo}"/></radialGradient>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>', f'<rect width="{W}" height="{H}" filter="url(#g)"/>']
    g = [f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha}" stroke-linecap="round">']
    # Vitruvian study: circle and square with proportion lines (top right)
    cx, cy, r = 1340, 300, 190
    g.append(f'<circle cx="{cx}" cy="{cy}" r="{r}" stroke-width="2"/>')
    g.append(f'<rect x="{cx-165}" y="{cy-130}" width="330" height="330" stroke-width="1.6"/>')
    for k in range(0, 360, 30):
        a = math.radians(k)
        g.append(f'<line x1="{cx}" y1="{cy+40}" x2="{f(cx+r*math.cos(a))}" y2="{f(cy+r*math.sin(a))}" stroke-width=".7" stroke-dasharray="3 6"/>')
    # a figure suggested by simple strokes
    g.append(f'<circle cx="{cx}" cy="{cy-95}" r="26" stroke-width="1.6"/>')
    g.append(f'<path d="M{cx} {cy-69} L{cx} {cy+70} M{cx-160} {cy-40} L{cx+160} {cy-40} M{cx} {cy+70} L{cx-70} {cy+195} M{cx} {cy+70} L{cx+70} {cy+195}" stroke-width="1.8"/>')
    # flying machine wing (bottom left): ribs and membrane
    ox, oy = 120, 760
    for i in range(9):
        t = i / 8
        g.append(f'<path d="M{ox} {oy} Q{f(ox+160+t*90)} {f(oy-200+t*120)} {f(ox+330+t*60)} {f(oy-40+t*130)}" stroke-width="{1.4-t*0.6:.2f}"/>')
    g.append(f'<path d="M{ox+330} {oy-40} C{ox+360} {oy+30} {ox+380} {oy+70} {ox+390} {oy+90}" stroke-width="1.2"/>')
    # gears (bottom right)
    for gx, gy, gr, n in [(1380, 820, 70, 16), (1490, 760, 44, 11)]:
        pts = []
        for k in range(n * 2):
            a = math.pi * k / n
            rr = gr + (10 if k % 2 == 0 else 0)
            pts.append(f'{f(gx+rr*math.cos(a))},{f(gy+rr*math.sin(a))}')
        g.append(f'<polygon points="{" ".join(pts)}" stroke-width="1.3"/><circle cx="{gx}" cy="{gy}" r="{gr*0.35:.0f}" stroke-width="1"/>')
    # mirror script: wavy lines of "handwriting" in the margins
    for row in range(14):
        y = 90 + row * 30
        x0 = 60 + random.randint(0, 30)
        segs = []
        x = x0
        while x < x0 + 300 + random.randint(-40, 60):
            w = random.randint(14, 40)
            segs.append(f'M{x} {y} q{w/4:.0f} {-random.randint(4,9)} {w/2:.0f} 0 t{w/2:.0f} 0')
            x += w + random.randint(6, 12)
        g.append(f'<path d="{" ".join(segs)}" stroke-width=".9"/>')
    # anatomical study: a few concentric arcs (left middle)
    for k in range(6):
        g.append(f'<path d="M80 {470+k*12} A260 {180-k*20} 0 0 1 520 {470+k*12}" stroke-width=".8" stroke-dasharray="{2+k} 5"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Oxford library: oak shelves, leather spines, arched window light ---
def library(wall_hi, wall_lo, wood, spines, glow, alpha, lamp=1.0):
    d = grain('g', 0.6, 0.08, (0.2, 0.1, 0.02)) + (
        f'<linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{wall_hi}"/><stop offset="1" stop-color="{wall_lo}"/></linearGradient>'
        f'<radialGradient id="lamp" cx="50%" cy="50%" r="50%"><stop offset="0" stop-color="{glow}" stop-opacity=".55"/><stop offset="1" stop-color="{glow}" stop-opacity="0"/></radialGradient>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#w)"/>']
    # arched window in the middle top, very faint
    b.append(f'<path d="M700 420 V170 A100 100 0 0 1 900 170 V420 Z" fill="{glow}" fill-opacity="{alpha*0.35:.2f}"/>')
    b.append(f'<path d="M800 70 V420 M700 260 H900" stroke="{wood}" stroke-opacity="{alpha*0.6:.2f}" stroke-width="6"/>')
    # shelves on both sides: rows of spines
    for side in (0, 1):
        x0, x1 = (0, 560) if side == 0 else (1040, 1600)
        for row in range(6):
            y = 60 + row * 160
            b.append(f'<rect x="{x0}" y="{y+128}" width="{x1-x0}" height="14" fill="{wood}" fill-opacity="{alpha:.2f}"/>')
            x = x0 + 8
            while x < x1 - 20:
                w = random.randint(14, 30)
                h = random.randint(90, 126)
                c = random.choice(spines)
                b.append(f'<rect x="{x}" y="{y+128-h}" width="{w}" height="{h}" rx="2" fill="{c}" fill-opacity="{alpha:.2f}"/>')
                if random.random() < 0.5:
                    b.append(f'<rect x="{x+2}" y="{y+128-h+14}" width="{w-4}" height="3" fill="{glow}" fill-opacity="{alpha*0.8:.2f}"/>')
                x += w + random.choice([1, 1, 2, 6])
        # side uprights
        for ux in ((x0, x1 - 16) if side == 0 else (x0, x1 - 16)):
            b.append(f'<rect x="{ux}" y="0" width="16" height="{H}" fill="{wood}" fill-opacity="{alpha:.2f}"/>')
    b.append(f'<g class="lamp"><ellipse cx="300" cy="120" rx="320" ry="260" fill="url(#lamp)" opacity="{lamp}"/><ellipse cx="1300" cy="120" rx="320" ry="260" fill="url(#lamp)" opacity="{lamp}"/></g>')
    b.append(f'<rect width="{W}" height="{H}" filter="url(#g)"/>')
    return svg(''.join(b), d)

# --- Workshop blueprint: grid, technical drawing, dimension lines ---
def blueprint(bg_hi, bg_lo, line, alpha):
    d = (f'<linearGradient id="b" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></linearGradient>'
         f'<pattern id="fine" width="20" height="20" patternUnits="userSpaceOnUse"><path d="M20 0H0V20" fill="none" stroke="{line}" stroke-opacity="{alpha*0.35:.2f}" stroke-width=".6"/></pattern>'
         f'<pattern id="major" width="100" height="100" patternUnits="userSpaceOnUse"><path d="M100 0H0V100" fill="none" stroke="{line}" stroke-opacity="{alpha*0.7:.2f}" stroke-width="1"/></pattern>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#b)"/>', f'<rect width="{W}" height="{H}" fill="url(#fine)"/>', f'<rect width="{W}" height="{H}" fill="url(#major)"/>']
    g = [f'<g fill="none" stroke="{line}" stroke-opacity="{alpha:.2f}" stroke-width="1.5">']
    # a bracket part in plan view (right)
    g.append('<path d="M1180 640 h260 v-80 a60 60 0 0 0 -60 -60 h-120 v-120 h-80 Z"/>')
    g.append('<circle cx="1220" cy="600" r="18"/><circle cx="1400" cy="600" r="18"/><circle cx="1220" cy="430" r="14"/>')
    # dimension lines with arrows
    g.append(f'<path d="M1180 690 h260 M1180 680 v20 M1440 680 v20" stroke-width="1"/><path d="M1180 690 l10 -4 v8 Z M1440 690 l-10 -4 v8 Z" fill="{line}" stroke="none"/>')
    g.append('<path d="M1130 640 v-260 M1120 640 h20 M1120 380 h20" stroke-width="1"/>')
    # a gear elevation (left top)
    gx, gy = 260, 260
    for rr in (150, 118, 40):
        g.append(f'<circle cx="{gx}" cy="{gy}" r="{rr}"/>')
    for k in range(24):
        a = 2 * math.pi * k / 24
        g.append(f'<line x1="{f(gx+150*math.cos(a))}" y1="{f(gy+150*math.sin(a))}" x2="{f(gx+170*math.cos(a))}" y2="{f(gy+170*math.sin(a))}"/>')
    g.append(f'<path d="M{gx-200} {gy} H{gx+200} M{gx} {gy-200} V{gy+200}" stroke-dasharray="14 6 3 6" stroke-width="1"/>')
    # compass arcs and a title block (bottom right)
    g.append('<path d="M700 900 A220 220 0 0 1 1100 900" stroke-dasharray="6 6"/>')
    g.append('<rect x="1300" y="860" width="260" height="100"/><path d="M1300 895 H1560 M1430 895 V960"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Garden (habits): vines, leaves, morning sun ---
def garden(bg_hi, bg_lo, leaf, stem, alpha):
    d = (f'<linearGradient id="s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></linearGradient>'
         f'<symbol id="leaf" viewBox="-20 -40 40 40"><path d="M0 0 C-18 -12 -14 -32 0 -40 C14 -32 18 -12 0 0 Z M0 0 V-36" /></symbol>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>', f'<circle cx="1380" cy="170" r="120" fill="{leaf}" fill-opacity="{alpha*0.25:.2f}"/>']
    g = [f'<g fill="{leaf}" fill-opacity="{alpha:.2f}" stroke="{stem}" stroke-opacity="{alpha:.2f}" stroke-width="1.4">']
    for vine in range(5):
        x = random.choice([40, 120, 1480, 1560, 820])
        y = H
        pts = [(x, y)]
        for k in range(10):
            x += random.randint(-40, 40)
            y -= random.randint(50, 90)
            pts.append((x, y))
        d_ = 'M' + ' L'.join(f'{px} {py}' for px, py in pts)
        g.append(f'<path d="{d_}" fill="none" stroke-width="2"/>')
        for (px, py) in pts[1:]:
            rot = random.randint(-70, 70)
            s = random.uniform(0.7, 1.3)
            g.append(f'<use href="#leaf" x="{px-20}" y="{py-40}" width="40" height="40" transform="rotate({rot} {px} {py}) scale(1)" />')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Summit (goals): topographic contour lines ---
def contours(bg_hi, bg_lo, line, alpha):
    d = f'<linearGradient id="s" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></linearGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>']
    g = [f'<g fill="none" stroke="{line}" stroke-opacity="{alpha:.2f}">']
    for cx, cy, n in [(1250, 330, 14), (300, 820, 10)]:
        for k in range(1, n + 1):
            pts = []
            for j in range(73):
                a = 2 * math.pi * j / 72
                rr = k * 26 * (1 + 0.18 * math.sin(3 * a + k * 0.4) + 0.08 * math.sin(7 * a))
                pts.append(f'{f(cx+rr*math.cos(a)*1.4)},{f(cy+rr*math.sin(a))}')
            g.append(f'<polygon points="{" ".join(pts)}" stroke-width="{1.6 if k % 5 == 0 else 0.8}"/>')
    g.append(f'<path d="M1250 330 l-14 26 h28 Z" fill="{line}" fill-opacity="{alpha:.2f}"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Almanac (calendar): star chart with constellation lines ---
def almanac(bg_hi, bg_lo, star, alpha):
    d = f'<radialGradient id="s" cx="50%" cy="0%" r="110%"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>']
    g = [f'<g fill="{star}" fill-opacity="{alpha:.2f}" stroke="{star}" stroke-opacity="{alpha*0.6:.2f}">']
    for k in range(160):
        x, y = random.randint(0, W), random.randint(0, H)
        r = random.choice([0.8, 1, 1.2, 1.6, 2.2])
        g.append(f'<circle class="star" cx="{x}" cy="{y}" r="{r}" stroke="none"/>')
    for cons in range(5):
        x, y = random.randint(100, W - 300), random.randint(80, H - 200)
        pts = [(x, y)]
        for k in range(5):
            x += random.randint(30, 110); y += random.randint(-70, 70)
            pts.append((x, y))
        g.append('<path fill="none" stroke-width=".8" d="M' + ' L'.join(f'{a} {c}' for a, c in pts) + '"/>')
        for a, c in pts:
            g.append(f'<circle cx="{a}" cy="{c}" r="2.6" stroke="none"/>')
    g.append(f'<circle cx="800" cy="1500" r="900" fill="none" stroke-dasharray="2 10"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Ledger (budget): green-ruled accounting paper ---
def ledger(bg, line, red, alpha):
    d = (f'<pattern id="rows" width="{W}" height="28" patternUnits="userSpaceOnUse"><path d="M0 27.5 H{W}" stroke="{line}" stroke-opacity="{alpha*0.6:.2f}"/></pattern>')
    b = [f'<rect width="{W}" height="{H}" fill="{bg}"/>', f'<rect width="{W}" height="{H}" fill="url(#rows)"/>']
    for x in (110, 1180, 1290, 1400, 1510):
        b.append(f'<path d="M{x} 0 V{H}" stroke="{line}" stroke-opacity="{alpha:.2f}" stroke-width="1.2"/><path d="M{x+4} 0 V{H}" stroke="{line}" stroke-opacity="{alpha*0.6:.2f}"/>')
    b.append(f'<path d="M96 0 V{H}" stroke="{red}" stroke-opacity="{alpha:.2f}" stroke-width="1.5"/>')
    return svg(''.join(b), d)

# --- Atlas (areas, archive): old map with a compass rose and meridians ---
def atlas(bg_hi, bg_lo, ink, alpha):
    d = grain('g', 0.7, 0.07, (0.3, 0.2, 0.1)) + f'<radialGradient id="p" cx="50%" cy="50%" r="80%"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>', f'<rect width="{W}" height="{H}" filter="url(#g)"/>']
    g = [f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha:.2f}">']
    for k in range(-6, 7):
        g.append(f'<path d="M{800+k*130} 0 Q{800+k*170} 500 {800+k*130} {H}" stroke-width=".7"/>')
    for k in range(1, 7):
        g.append(f'<path d="M0 {k*140} Q800 {k*140-40} {W} {k*140}" stroke-width=".7"/>')
    cx, cy = 1360, 780
    for k in range(16):
        a = math.pi * k / 8
        rr = 120 if k % 4 == 0 else (80 if k % 2 == 0 else 50)
        g.append(f'<path d="M{cx} {cy} L{f(cx+rr*math.cos(a))} {f(cy+rr*math.sin(a))}" stroke-width="{1.6 if k % 4 == 0 else 0.8}"/>')
    g.append(f'<circle cx="{cx}" cy="{cy}" r="60" stroke-width="1"/><circle cx="{cx}" cy="{cy}" r="90" stroke-dasharray="2 4"/>')
    # a coastline
    x, y, pts = 0, 300, []
    while x < 700:
        pts.append(f'{x} {y}'); x += random.randint(15, 40); y += random.randint(-25, 25)
    g.append('<path stroke-width="1.4" d="M' + ' L'.join(pts) + '"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Desk (tasks): dotted bullet-journal paper ---
def desk(bg, dot, alpha):
    d = f'<pattern id="dots" width="24" height="24" patternUnits="userSpaceOnUse"><circle cx="12" cy="12" r="1.2" fill="{dot}" fill-opacity="{alpha:.2f}"/></pattern>'
    return svg(f'<rect width="{W}" height="{H}" fill="{bg}"/><rect width="{W}" height="{H}" fill="url(#dots)"/>', d)

art = {
  'notebook-light': notebook('#5b3a1a', '#f7ecd2', '#e9d4a8', 0.30),
  'notebook-dark': notebook('#d9b77c', '#2b2318', '#1a140d', 0.22),
  'library-light': library('#f4ece0', '#e6d6bf', '#7a5332', ['#6e2a2a', '#2f4a3a', '#283c5c', '#7a5a2a', '#4a2f45'], '#f2c46d', 0.20),
  'library-dark': library('#24170f', '#140c07', '#3a2414', ['#5a1f1f', '#1f3a2c', '#1d2c48', '#5a4220', '#3a2236'], '#f2b04a', 0.55, 0.3),
  'blueprint-light': blueprint('#eef4fb', '#e2ecf8', '#1d5fae', 0.30),
  'blueprint-dark': blueprint('#0d2a4f', '#081c36', '#9cc4f2', 0.28),
  'garden-light': garden('#f2f8ee', '#e4f0dc', '#3f8a4a', '#2f6b3a', 0.20),
  'garden-dark': garden('#0f1a12', '#0a120c', '#5fae6a', '#3f8a4a', 0.25),
  'summit-light': contours('#f3f5f8', '#e7ecf2', '#56708f', 0.28),
  'summit-dark': contours('#111823', '#0b1018', '#7f9cc0', 0.25),
  'almanac-light': almanac('#f2f3fa', '#e3e6f4', '#3a4a8a', 0.30),
  'almanac-dark': almanac('#141a33', '#070a17', '#e8ecff', 0.60),
  'ledger-light': ledger('#f6faf3', '#3e8a5a', '#c0392b', 0.30),
  'ledger-dark': ledger('#0e1611', '#5fae7a', '#e06a5a', 0.25),
  'atlas-light': atlas('#f6efdf', '#eadfc4', '#6b4f2a', 0.24),
  'atlas-dark': atlas('#211c14', '#15110b', '#cfae7a', 0.20),
  'desk-light': desk('#f7f7f5', '#5d5d63', 0.35),
  'desk-dark': desk('#141416', '#a0a0a8', 0.25),
}
for name, s in art.items():
    open(os.path.join(out, name + '.svg'), 'w').write(s)
print('\n'.join(f'{n}: {len(s)//1024} KB' for n, s in art.items()))
