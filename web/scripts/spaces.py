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


# --- Diary (journal): a leaded window's evening light on a writing desk, after Vermeer ---
def diary(wall_hi, wall_lo, lead, light, ink, alpha):
    d = (f'<linearGradient id="w" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="{wall_hi}"/><stop offset="1" stop-color="{wall_lo}"/></linearGradient>'
         f'<linearGradient id="beam" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="{light}" stop-opacity="{alpha*1.4:.2f}"/><stop offset="1" stop-color="{light}" stop-opacity="0"/></linearGradient>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#w)"/>']
    # the window, top left: a frame of leaded diamond panes
    x0, y0, ww, wh = 70, 60, 330, 520
    b.append(f'<rect x="{x0}" y="{y0}" width="{ww}" height="{wh}" fill="{light}" fill-opacity="{alpha*0.9:.2f}"/>')
    g = [f'<g fill="none" stroke="{lead}" stroke-opacity="{alpha*1.6:.2f}">']
    g.append(f'<rect x="{x0}" y="{y0}" width="{ww}" height="{wh}" stroke-width="10"/><path d="M{x0+ww/2} {y0} V{y0+wh} M{x0} {y0+wh/2} H{x0+ww}" stroke-width="6"/>')
    step = 40
    for k in range(-14, 22):
        g.append(f'<path d="M{x0+k*step} {y0} l{wh} {wh}" stroke-width="1"/><path d="M{x0+k*step} {y0} l{-wh} {wh}" stroke-width="1"/>')
    g.append('</g>')
    clip = f'<clipPath id="win"><rect x="{x0}" y="{y0}" width="{ww}" height="{wh}"/></clipPath>'
    d += clip
    g = [g[0].replace('<g ', '<g clip-path="url(#win)" ', 1)] + g[1:]
    # the light falling across the room onto the desk
    b.append(f'<path d="M{x0+ww} {y0} L{x0+ww} {y0+wh} L1250 {H} L760 {H} Z" fill="url(#beam)"/>')
    # the desk and an open diary with lines of writing
    b.append(f'<path d="M0 760 H{W}" stroke="{ink}" stroke-opacity="{alpha*1.2:.2f}" stroke-width="3"/>')
    bx, by = 980, 690
    b.append(f'<path d="M{bx} {by} q160 -40 300 0 q140 -40 300 0 v60 q-150 -30 -300 0 q-150 -30 -300 0 Z" fill="{wall_hi}" fill-opacity=".55" stroke="{ink}" stroke-opacity="{alpha*1.3:.2f}"/>')
    for row in range(4):
        y = by + 14 + row * 11
        b.append(f'<path d="M{bx+30} {y} h{220-row*20} M{bx+330} {y} h{200-row*30}" stroke="{ink}" stroke-opacity="{alpha:.2f}" stroke-dasharray="14 6"/>')
    # a quill in its inkwell
    b.append(f'<path d="M1500 740 q-30 -120 30 -240 q-10 120 -20 240" fill="{ink}" fill-opacity="{alpha*0.8:.2f}"/><rect x="1480" y="735" width="44" height="25" rx="6" fill="{ink}" fill-opacity="{alpha*1.2:.2f}"/>')
    return svg(''.join(b + g), d)

# --- Observatory (insights): star trails around the pole, Galileo's moons ---
def observatory(sky_hi, sky_lo, line, alpha):
    d = f'<radialGradient id="s" cx="85%" cy="5%" r="120%"><stop offset="0" stop-color="{sky_hi}"/><stop offset="1" stop-color="{sky_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>']
    g = [f'<g fill="none" stroke="{line}" stroke-opacity="{alpha:.2f}" stroke-linecap="round">']
    px, py = 1380, 90
    for k in range(70):
        r = random.randint(60, 1500)
        a0 = random.uniform(0, 2 * math.pi)
        a1 = a0 + random.uniform(0.25, 0.6)
        g.append(f'<path d="M{f(px+r*math.cos(a0))} {f(py+r*math.sin(a0))} A{r} {r} 0 0 1 {f(px+r*math.cos(a1))} {f(py+r*math.sin(a1))}" stroke-width="{random.choice([0.8, 1.1, 1.5])}"/>')
    # Galileo's sketches of the moon: a row of phases
    for i in range(6):
        cx, cy, r = 120 + i * 95, 130, 34
        k = math.cos(math.pi * (i + 1) / 7)
        g.append(f'<circle cx="{cx}" cy="{cy}" r="{r}" stroke-width="1.2"/>')
        g.append(f'<path d="M{cx} {cy-r} A{r} {r} 0 0 1 {cx} {cy+r} A{f(abs(k)*r)} {r} 0 0 {0 if k > 0 else 1} {cx} {cy-r} Z" fill="{line}" fill-opacity="{alpha*0.7:.2f}" stroke="none"/>')
    g.append('</g>')
    # the observatory: a dome with its slit open, on a hill
    b.append(f'<path d="M0 {H} V880 Q400 820 800 860 T{W} 850 V{H} Z" fill="{line}" fill-opacity="{alpha*0.5:.2f}"/>')
    b.append(f'<path d="M180 870 V780 A130 130 0 0 1 440 780 V870 Z" fill="{line}" fill-opacity="{alpha*0.9:.2f}"/>')
    b.append(f'<path d="M300 650 L340 650 L330 780 L310 780 Z" fill="{sky_lo}"/><path d="M318 700 L470 560" stroke="{line}" stroke-opacity="{alpha*1.4:.2f}" stroke-width="14" stroke-linecap="round"/>')
    return svg(''.join(b + g), d)

# --- Lighthouse (weekly review): a headland and its beam over the sea, after Hopper ---
def lighthouse(sky_hi, sky_lo, sea, rock, tower, beam, alpha):
    d = (f'<linearGradient id="s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{sky_hi}"/><stop offset="1" stop-color="{sky_lo}"/></linearGradient>'
         f'<linearGradient id="b" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="{beam}" stop-opacity="{alpha:.2f}"/><stop offset="1" stop-color="{beam}" stop-opacity="0"/></linearGradient>')
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>']
    b.append(f'<rect y="700" width="{W}" height="300" fill="{sea}" fill-opacity="{alpha*1.2:.2f}"/>')
    for k in range(12):
        y = 720 + k * 22
        b.append(f'<path d="M{random.randint(0, 300)} {y} q40 -6 80 0 t80 0 t80 0" fill="none" stroke="{rock}" stroke-opacity="{alpha*0.6:.2f}"/>'.replace('M', f'M', 1))
        b.append(f'<path d="M{random.randint(500, 1500)} {y} q40 -6 80 0 t80 0" fill="none" stroke="{rock}" stroke-opacity="{alpha*0.6:.2f}"/>')
    # the beam, the headland, the tower and its keeper's house
    b.append(f'<path d="M1240 380 L{W} 250 L{W} 470 Z" fill="url(#b)"/><path d="M1240 380 L700 300 L700 420 Z" fill="url(#b)" transform="rotate(180 1240 380) scale(1 1)" opacity=".5"/>')
    b.append(f'<path d="M900 {H} L980 700 Q1100 620 1240 640 Q1420 660 {W} 700 V{H} Z" fill="{rock}" fill-opacity="{alpha*1.6:.2f}"/>')
    b.append(f'<path d="M1210 640 L1222 420 L1258 420 L1270 640 Z" fill="{tower}" fill-opacity="{alpha*2:.2f}"/><rect x="1216" y="380" width="48" height="40" rx="4" fill="{beam}" fill-opacity="{min(1, alpha*3):.2f}"/><path d="M1206 380 L1240 340 L1274 380 Z" fill="{tower}" fill-opacity="{alpha*2:.2f}"/>')
    b.append(f'<path d="M1290 640 V590 L1330 560 L1370 590 V640 Z" fill="{tower}" fill-opacity="{alpha*1.6:.2f}"/>')
    return svg(''.join(b), d)

# --- Automata (agents): clockwork gears and cams, after Jaquet-Droz's Writer ---
def automata(bg_hi, bg_lo, metal, alpha):
    d = f'<radialGradient id="s" cx="30%" cy="30%" r="100%"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#s)"/>']
    g = [f'<g fill="none" stroke="{metal}" stroke-opacity="{alpha:.2f}">']
    for gx, gy, gr, n in [(1300, 300, 170, 28), (1500, 520, 100, 18), (1160, 600, 90, 16), (1400, 820, 130, 22), (160, 860, 120, 20), (330, 760, 60, 12)]:
        pts = []
        for k in range(n * 2):
            a = math.pi * k / n
            rr = gr + (14 if k % 2 == 0 else 0)
            pts.append(f'{f(gx+rr*math.cos(a))},{f(gy+rr*math.sin(a))}')
        g.append(f'<polygon points="{" ".join(pts)}" stroke-width="1.6"/><circle cx="{gx}" cy="{gy}" r="{gr*0.22:.0f}" stroke-width="1.2"/>')
        for k in range(5):
            a = 2 * math.pi * k / 5
            g.append(f'<path d="M{f(gx+gr*0.22*math.cos(a))} {f(gy+gr*0.22*math.sin(a))} L{f(gx+gr*0.8*math.cos(a))} {f(gy+gr*0.8*math.sin(a))}" stroke-width="1"/>')
    # a cam stack and its follower: the Writer's memory of each letter
    for k in range(6):
        g.append(f'<ellipse cx="900" cy="{180+k*26}" rx="{70+k*8}" ry="10" stroke-width="1"/>')
    g.append(f'<path d="M980 180 L1060 140 L1100 330" stroke-width="2"/><path d="M120 120 q60 -40 120 0 t120 0 t120 0" stroke-dasharray="4 6"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Signal (tech watch): a radio dish listening to signals ---
def signal(bg_hi, bg_lo, ink, signal, alpha):
    d = grain('g', 0.7, 0.06, (0.1, 0.15, 0.3)) + f'<radialGradient id="p" cx="70%" cy="20%" r="90%"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>', f'<rect width="{W}" height="{H}" filter="url(#g)"/>']
    g = [f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha:.2f}" stroke-linecap="round">']
    # the dish (bottom right): a parabola on a mount, its feed, its struts
    cx, cy = 1290, 760
    g.append(f'<path d="M{cx-190} {cy-150} Q{cx-20} {cy+90} {cx+170} {cy-170}" stroke-width="2.4"/>')
    g.append(f'<path d="M{cx-190} {cy-150} L{cx+170} {cy-170}" stroke-width="1" stroke-dasharray="4 6"/>')
    g.append(f'<path d="M{cx-12} {cy-25} L{cx+40} {cy-210} M{cx-120} {cy-110} L{cx+40} {cy-210} M{cx+110} {cy-125} L{cx+40} {cy-210}" stroke-width="1.2"/>')
    g.append(f'<circle cx="{cx+40}" cy="{cy-210}" r="9" stroke-width="1.6"/>')
    g.append(f'<path d="M{cx-12} {cy-25} L{cx-60} {cy+180} M{cx-12} {cy-25} L{cx+50} {cy+180} M{cx-90} {cy+180} L{cx+90} {cy+180}" stroke-width="2"/>')
    # signals arriving: arcs from the top left
    for k in range(9):
        r = 120 + k * 95
        g.append(f'<path d="M{f(160+r*math.cos(math.radians(10)))} {f(120+r*math.sin(math.radians(10)))} A{r} {r} 0 0 1 {f(160+r*math.cos(math.radians(80)))} {f(120+r*math.sin(math.radians(80)))}" '
                 f'stroke="{signal}" stroke-width="{1.4-k*0.1:.2f}" stroke-dasharray="{3+k} {6+k}"/>')
    # a waveform across the middle: the news as a signal
    pts = []
    for x in range(0, W + 1, 8):
        y = 520 + 40 * math.sin(x / 70) * math.exp(-((x - 700) / 520) ** 2) + random.uniform(-4, 4)
        pts.append(f'{x} {f(y)}')
    g.append(f'<path d="M{" L".join(pts)}" stroke="{signal}" stroke-width="1.2"/>')
    g.append('</g>')
    # points of data: small dots scattered like received packets
    dots = [f'<circle cx="{random.randint(40, W-40)}" cy="{random.randint(40, 460)}" r="{random.choice([1.2, 1.6, 2.2])}" fill="{signal}" fill-opacity="{alpha*1.4:.2f}"/>' for _ in range(70)]
    return svg(''.join(b + g + dots), d)

# --- Constellation (news graph): a star chart with its grid ---
def constellation(bg_hi, bg_lo, ink, star, alpha):
    d = f'<radialGradient id="p" cx="50%" cy="40%" r="85%"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></radialGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>']
    g = [f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha:.2f}">']
    cx, cy = 800, 520
    for r in (180, 330, 480, 630, 780):
        g.append(f'<circle cx="{cx}" cy="{cy}" r="{r}" stroke-width=".7" stroke-dasharray="2 7"/>')
    for k in range(12):
        a = math.pi * k / 6
        g.append(f'<line x1="{cx}" y1="{cy}" x2="{f(cx+900*math.cos(a))}" y2="{f(cy+900*math.sin(a))}" stroke-width=".5"/>')
    g.append(f'<path d="M0 {cy+160} Q{cx} {cy-260} {W} {cy+120}" stroke-width="1" stroke-dasharray="10 8"/>')  # the ecliptic
    g.append('</g>')
    # constellations: a few groups of stars joined by lines
    groups = []
    for gx, gy in [(260, 230), (1270, 210), (1180, 780), (360, 760), (820, 330)]:
        pts = [(gx + random.randint(-150, 150), gy + random.randint(-110, 110)) for _ in range(random.randint(4, 6))]
        pts.sort()
        groups.append(f'<path d="M{" L".join(f"{x} {y}" for x, y in pts)}" fill="none" stroke="{star}" stroke-opacity="{alpha*1.6:.2f}" stroke-width="1"/>')
        for x, y in pts:
            groups.append(f'<circle cx="{x}" cy="{y}" r="{random.choice([2.5, 3.2, 4])}" fill="{star}" fill-opacity="{min(1, alpha*3):.2f}"/>')
    field = [f'<circle cx="{random.randint(0, W)}" cy="{random.randint(0, H)}" r="{random.choice([0.8, 1.1, 1.5])}" fill="{star}" fill-opacity="{alpha*1.8:.2f}"/>' for _ in range(160)]
    return svg(''.join(b + g + field + groups), d)

# --- Studio (podcast): a waveform, a microphone, sound rings ---
def studio(bg_hi, bg_lo, ink, wave, alpha):
    d = grain('g', 0.75, 0.05, (0.25, 0.1, 0.2)) + f'<linearGradient id="p" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></linearGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>', f'<rect width="{W}" height="{H}" filter="url(#g)"/>']
    g = []
    # the waveform: rounded bars along the bottom third
    for i, x in enumerate(range(40, W - 30, 22)):
        h = 18 + 150 * abs(math.sin(i / 5.3) * math.sin(i / 2.1 + 1)) * (0.4 + 0.6 * math.exp(-((x - 1100) / 600) ** 2))
        g.append(f'<rect x="{x}" y="{f(760-h/2)}" width="9" height="{f(h)}" rx="4.5" fill="{wave}" fill-opacity="{alpha:.2f}"/>')
    # a microphone (top right) and its rings
    mx, my = 1300, 300
    g.append(f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha*1.5:.2f}" stroke-linecap="round">')
    g.append(f'<rect x="{mx-46}" y="{my-120}" width="92" height="170" rx="46" stroke-width="2.4"/>')
    for k in range(5):
        g.append(f'<line x1="{mx-46}" y1="{my-80+k*24}" x2="{mx+46}" y2="{my-80+k*24}" stroke-width=".8"/>')
    g.append(f'<path d="M{mx-80} {my+10} Q{mx-80} {my+110} {mx} {my+110} Q{mx+80} {my+110} {mx+80} {my+10}" stroke-width="2"/>')
    g.append(f'<path d="M{mx} {my+110} L{mx} {my+190} M{mx-60} {my+190} L{mx+60} {my+190}" stroke-width="2"/>')
    for k in range(4):
        r = 170 + k * 70
        g.append(f'<path d="M{f(mx-r*0.8)} {f(my-r*0.6)} A{r} {r} 0 0 0 {f(mx-r*0.8)} {f(my+r*0.6)}" stroke-width="1.2" stroke-dasharray="{4+k*2} {8+k*2}"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

# --- Dispatch (digest): paper planes on their routes at sunrise ---
def dispatch(bg_hi, bg_lo, sun, ink, alpha):
    d = f'<linearGradient id="p" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{bg_hi}"/><stop offset="1" stop-color="{bg_lo}"/></linearGradient>'
    b = [f'<rect width="{W}" height="{H}" fill="url(#p)"/>']
    # the sun rising behind the horizon (bottom left), its rays
    sx, sy = 300, 1000
    b.append(f'<circle cx="{sx}" cy="{sy}" r="260" fill="{sun}" fill-opacity="{alpha*0.9:.2f}"/>')
    for k in range(13):
        a = math.pi + math.pi * k / 12
        b.append(f'<line x1="{f(sx+300*math.cos(a))}" y1="{f(sy+300*math.sin(a))}" x2="{f(sx+620*math.cos(a))}" y2="{f(sy+620*math.sin(a))}" stroke="{sun}" stroke-opacity="{alpha*0.8:.2f}" stroke-width="3" stroke-linecap="round"/>')
    g = [f'<g fill="none" stroke="{ink}" stroke-opacity="{alpha:.2f}" stroke-linecap="round" stroke-linejoin="round">']
    planes = [(520, 640, 1, 1.0), (980, 330, -1, 1.3), (1360, 560, 1, 0.9), (760, 180, 1, 0.7)]
    for px, py, flip, sc in planes:
        # the route: a dashed loop behind each plane
        g.append(f'<path d="M{f(px-flip*360*sc)} {f(py+160*sc)} C{f(px-flip*240*sc)} {f(py-60*sc)} {f(px-flip*140*sc)} {f(py+120*sc)} {f(px-flip*40*sc)} {f(py+10*sc)}" stroke-width="1.4" stroke-dasharray="6 9"/>')
        tip = (px + flip * 70 * sc, py - 30 * sc)
        g.append(f'<path d="M{f(px-flip*40*sc)} {f(py+10*sc)} L{f(tip[0])} {f(tip[1])} L{f(px-flip*20*sc)} {f(py+40*sc)} Z M{f(tip[0])} {f(tip[1])} L{f(px)} {f(py+18*sc)} L{f(px-flip*20*sc)} {f(py+40*sc)}" stroke-width="2"/>')
    # an envelope (top left)
    ex, ey = 140, 150
    g.append(f'<rect x="{ex}" y="{ey}" width="190" height="120" rx="6" stroke-width="2"/><path d="M{ex} {ey} L{ex+95} {ey+70} L{ex+190} {ey}" stroke-width="2"/>')
    g.append('</g>')
    return svg(''.join(b + g), d)

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
  'diary-light': diary('#f3eadb', '#e6d8c0', '#4a3a28', '#fff2c8', '#3a2a18', 0.22),
  'diary-dark': diary('#1d1812', '#120e0a', '#c9a96e', '#f2c46d', '#d9b77c', 0.20),
  'observatory-light': observatory('#e9edf7', '#d9e0f0', '#2f3f73', 0.30),
  'observatory-dark': observatory('#18203f', '#05070f', '#dfe6ff', 0.45),
  'lighthouse-light': lighthouse('#eef3f7', '#dde8f0', '#5b86a8', '#3c5568', '#2a3a48', '#ffd27a', 0.22),
  'lighthouse-dark': lighthouse('#0f1a2a', '#08101c', '#2c4a68', '#9fb4c8', '#c8d4e0', '#ffd27a', 0.28),
  'automata-light': automata('#f4efe6', '#e6dccb', '#6b5434', 0.26),
  'automata-dark': automata('#1c1712', '#0f0c09', '#d4b483', 0.24),
  'signal-light': signal('#f1f5fb', '#e3eaf5', '#2c4a6e', '#1f7a8c', 0.30),
  'signal-dark': signal('#13203a', '#080d19', '#9fb8dc', '#4fc3d0', 0.30),
  'constellation-light': constellation('#f2f1fa', '#e2e0f2', '#4a4a8a', '#3b3b7a', 0.22),
  'constellation-dark': constellation('#141433', '#05050f', '#8f8fd8', '#e8e8ff', 0.30),
  'studio-light': studio('#fbf3f6', '#f0e2ea', '#6a2e4f', '#b0447a', 0.22),
  'studio-dark': studio('#22121c', '#0f070c', '#e2a8c6', '#e06aa4', 0.28),
  'dispatch-light': dispatch('#fdf6ec', '#f4e6d2', '#f2a65a', '#7a4a20', 0.30),
  'dispatch-dark': dispatch('#1d1622', '#0d0a10', '#e07a3a', '#e8c9a0', 0.26),
}
for name, s in art.items():
    open(os.path.join(out, name + '.svg'), 'w').write(s)
print('\n'.join(f'{n}: {len(s)//1024} KB' for n, s in art.items()))
