#!/usr/bin/env python3
"""refcompare: compare fork renders (harness/renderparity/out) against the Skia
reference renders (harness/renderparity/ref), scoring per-pixel deltas and
producing diff heatmap images for the worst regions."""
import os
import sys
import numpy as np
from PIL import Image

OUT = os.environ.get('PARITY_OUT', 'harness/renderparity/out')
REF = os.environ.get('PARITY_REF', 'harness/renderparity/ref')
# The heatmaps follow OUT, so an alternate run (the Android device renders, say)
# does not overwrite the desktop's diffs.
DIFF = os.environ.get('PARITY_DIFF', os.path.join(OUT, 'diff'))


def compare(name):
    out_p = os.path.join(OUT, name + '.png')
    ref_p = os.path.join(REF, name + '.png')
    if not (os.path.exists(out_p) and os.path.exists(ref_p)):
        return None
    out = np.asarray(Image.open(out_p).convert('RGB')).astype(int)
    ref = np.asarray(Image.open(ref_p).convert('RGB')).astype(int)
    h = min(out.shape[0], ref.shape[0])
    w = min(out.shape[1], ref.shape[1])
    out, ref = out[:h, :w], ref[:h, :w]
    diff = np.abs(out - ref).sum(axis=2)  # 0..765
    stats = {
        'name': name,
        'mean': diff.mean(),
        'pct_over60': (diff > 60).mean() * 100,
        'pct_over200': (diff > 200).mean() * 100,
    }
    # diff heatmap: red intensity = delta
    heat = np.zeros((h, w, 3), dtype=np.uint8)
    heat[..., 0] = np.clip(diff, 0, 255)
    heat[..., 1] = np.clip(diff // 3, 0, 255)
    Image.fromarray(heat).save(os.path.join(DIFF, name + '_diff.png'))
    # side-by-side stacked: fork / ref / heat
    combo = Image.new('RGB', (w, h * 3 + 8), 'black')
    combo.paste(Image.open(out_p).convert('RGB').crop((0, 0, w, h)), (0, 0))
    combo.paste(Image.open(ref_p).convert('RGB').crop((0, 0, w, h)), (0, h + 4))
    combo.paste(Image.fromarray(heat), (0, 2 * h + 8))
    combo.save(os.path.join(DIFF, name + '_stack.png'))
    return stats


def main():
    os.makedirs(DIFF, exist_ok=True)
    names = sys.argv[1:] if len(sys.argv) > 1 else [
        f[:-4] for f in os.listdir(REF) if f.endswith('.png')]
    rows = []
    for n in sorted(names):
        s = compare(n)
        if s:
            rows.append(s)
    rows.sort(key=lambda r: -r['mean'])
    print(f"{'name':24s} {'meanΔ':>7s} {'%>60':>6s} {'%>200':>6s}")
    for r in rows:
        print(f"{r['name']:24s} {r['mean']:7.1f} {r['pct_over60']:6.2f} {r['pct_over200']:6.2f}")


if __name__ == '__main__':
    main()
