"""Generate client lobby placeholder PNG assets aligned with Digital Actuarialism."""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

OUT = Path(__file__).resolve().parent.parent / "public" / "images" / "lobby"

PRIMARY = (0, 102, 255)
PRIMARY_DARK = (0, 80, 203)
SURFACE = (247, 249, 251)
WHITE = (255, 255, 255)
SLATE = (100, 116, 139)
SLATE_DARK = (51, 65, 85)
AMBER = (245, 158, 11)
EMERALD = (16, 185, 129)


def save(img: Image.Image, name: str) -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    path = OUT / name
    img.save(path, format="PNG", optimize=True)
    print(f"  {path.relative_to(OUT.parent.parent)}")


def rgba(size: tuple[int, int], color: tuple[int, ...] = (0, 0, 0, 0)) -> Image.Image:
    return Image.new("RGBA", size, color)


def rr(d: ImageDraw.ImageDraw, box: tuple[int, int, int, int], **kwargs) -> None:
    d.rounded_rectangle(list(box), **kwargs)


def gradient_bg(size: int, c1: tuple[int, int, int], c2: tuple[int, int, int]) -> Image.Image:
    img = Image.new("RGBA", (size, size))
    px = img.load()
    for y in range(size):
        t = y / max(size - 1, 1)
        r = int(c1[0] + (c2[0] - c1[0]) * t)
        g = int(c1[1] + (c2[1] - c1[1]) * t)
        b = int(c1[2] + (c2[2] - c1[2]) * t)
        for x in range(size):
            px[x, y] = (r, g, b, 255)
    return img


def draw_avatar() -> None:
    size = 256
    base = gradient_bg(size, (0, 118, 255), (0, 64, 180))
    draw = ImageDraw.Draw(base)
    draw.ellipse((68, 148, 188, 248), fill=(255, 255, 255, 220))
    draw.ellipse((88, 56, 168, 136), fill=(255, 255, 255, 235))
    save(base, "avatar-user.png")


def draw_icon_back() -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    d.polygon([(54, 24), (30, 48), (54, 72)], fill=PRIMARY)
    rr(d, (30, 42, 66, 54), radius=3, fill=PRIMARY)
    save(img, "icon-back.png")


def draw_icon_search() -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    d.ellipse((22, 22, 58, 58), outline=PRIMARY, width=6)
    d.line([(54, 54), (72, 72)], fill=PRIMARY, width=6)
    save(img, "icon-search.png")


def draw_notify() -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    d.polygon([(48, 18), (68, 42), (68, 58), (28, 58), (28, 42)], fill=PRIMARY)
    d.ellipse((38, 62, 58, 74), fill=PRIMARY)
    d.ellipse((62, 24, 72, 34), fill=(239, 68, 68, 255))
    save(img, "notify-placeholder.png")


def draw_announce() -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    d.polygon([(24, 34), (24, 62), (48, 48)], fill=PRIMARY)
    rr(d, (48, 30, 72, 66), radius=8, fill=PRIMARY_DARK)
    d.line([(72, 38), (82, 32), (82, 64), (72, 58)], fill=PRIMARY_DARK, width=4)
    save(img, "announce-placeholder.png")


def draw_tab(kind: str) -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    if kind == "lobby":
        rr(d, (20, 20, 76, 76), radius=14, fill=(239, 246, 255, 255))
        rr(d, (28, 28, 44, 44), radius=6, fill=PRIMARY)
        rr(d, (52, 28, 68, 44), radius=6, fill=PRIMARY)
        rr(d, (28, 52, 44, 68), radius=6, fill=PRIMARY)
        rr(d, (52, 52, 68, 68), radius=6, fill=PRIMARY)
        name = "tab-lobby.png"
    elif kind == "cloud":
        d.ellipse((18, 44, 44, 62), fill=PRIMARY)
        d.ellipse((34, 36, 62, 58), fill=PRIMARY)
        d.ellipse((52, 42, 78, 62), fill=PRIMARY)
        name = "tab-cloud.png"
    else:
        d.ellipse((36, 22, 60, 46), fill=PRIMARY)
        d.pieslice((24, 42, 72, 86), start=200, end=340, fill=PRIMARY)
        name = "tab-member.png"
    save(img, name)


def draw_bento(kind: str) -> None:
    if kind == "copy":
        size, name = 144, "bento-copy-hall.png"
    elif kind == "custom":
        size, name = 88, "bento-custom-scheme.png"
    else:
        size, name = 88, "bento-scheme-download.png"
    img = rgba((size, size))
    d = ImageDraw.Draw(img)
    pad = size // 6
    if kind == "copy":
        rr(d, (pad, pad * 2, size - pad, size - pad), radius=12, fill=(239, 246, 255, 255))
        pts = [(pad * 2, size - pad * 2), (size // 2, pad * 2), (size - pad * 2, size - pad * 2)]
        d.line(pts, fill=PRIMARY, width=8)
        d.ellipse((size - pad * 3, pad, size - pad, pad * 3), fill=(239, 68, 68, 255))
    elif kind == "custom":
        rr(d, (pad, pad, size - pad, size - pad), radius=10, outline=PRIMARY, width=5)
        d.line([(pad * 2, size // 2), (size - pad * 2, size // 2)], fill=AMBER, width=4)
        d.line([(size // 2, pad * 2), (size // 2, size - pad * 2)], fill=EMERALD, width=4)
    else:
        rr(d, (pad, pad, size - pad, size - pad), radius=10, fill=(239, 246, 255, 255))
        d.polygon(
            [
                (size // 2, pad * 2),
                (size - pad * 2, size // 2 + 4),
                (size // 2 + 4, size // 2 + 4),
                (size // 2 + 4, size - pad * 2),
                (pad * 2, size // 2 + 4),
            ],
            fill=PRIMARY,
        )
    save(img, name)


def draw_news_item() -> None:
    img = rgba((88, 88))
    d = ImageDraw.Draw(img)
    rr(d, (14, 16, 74, 72), radius=8, fill=WHITE)
    rr(d, (14, 16, 74, 72), radius=8, outline=PRIMARY, width=3)
    for y in (30, 42, 54):
        rr(d, (24, y, 64, y + 6), radius=3, fill=(203, 213, 225, 255))
    save(img, "news-item.png")


def draw_timer() -> None:
    img = rgba((72, 72))
    d = ImageDraw.Draw(img)
    d.ellipse((8, 8, 64, 64), outline=PRIMARY, width=5)
    d.line([(36, 36), (36, 22)], fill=PRIMARY, width=4)
    d.line([(36, 36), (50, 36)], fill=AMBER, width=4)
    save(img, "icon-timer.png")


def draw_drag_handle() -> None:
    img = rgba((112, 48))
    d = ImageDraw.Draw(img)
    for cx in (36, 56, 76):
        for cy in (16, 24, 32):
            d.ellipse((cx - 3, cy - 3, cx + 3, cy + 3), fill=SLATE)
    save(img, "icon-drag-handle.png")


def draw_icon_chevron() -> None:
    img = rgba((48, 48))
    d = ImageDraw.Draw(img)
    d.polygon([(12, 18), (24, 30), (36, 18)], fill=SLATE)
    save(img, "icon-chevron-down.png")


def draw_icon_filter() -> None:
    img = rgba((72, 72))
    d = ImageDraw.Draw(img)
    d.polygon([(16, 14), (56, 14), (44, 30), (44, 50), (28, 58), (28, 30)], fill=PRIMARY)
    save(img, "icon-filter.png")


def draw_scheme_card() -> None:
    img = rgba((96, 96))
    d = ImageDraw.Draw(img)
    rr(d, (16, 24, 80, 72), radius=8, fill=(239, 246, 255, 255))
    d.line([(24, 58), (38, 44), (52, 52), (72, 34)], fill=PRIMARY, width=5)
    d.ellipse((24, 34, 32, 42), fill=EMERALD)
    save(img, "icon-scheme.png")


def draw_feature_hero() -> None:
    w, h = 1200, 515
    img = Image.new("RGBA", (w, h), SURFACE)
    d = ImageDraw.Draw(img)
    for y in range(h):
        t = y / h
        c = (
            int(239 + (255 - 239) * t * 0.3),
            int(246 + (255 - 246) * t * 0.3),
            int(255 - 20 * t),
            255,
        )
        d.line([(0, y), (w, y)], fill=c)
    rr(d, (48, 48, w - 48, h - 48), radius=28, fill=(255, 255, 255, 230))
    rr(d, (80, 80, 420, h - 80), radius=20, fill=(239, 246, 255, 255))
    d.ellipse((140, 130, 360, 350), fill=PRIMARY)
    d.polygon([(420, 180), (420, 320), (520, 250)], fill=PRIMARY_DARK)
    try:
        font_l = ImageFont.truetype("arial.ttf", 52)
        font_s = ImageFont.truetype("arial.ttf", 28)
    except OSError:
        font_l = ImageFont.load_default()
        font_s = ImageFont.load_default()
    d.text((560, 120), "Platform Announcement", fill=SLATE_DARK, font=font_l)
    d.text((560, 210), "Stay updated with the latest features", fill=SLATE, font=font_s)
    rr(d, (560, 280, 760, 340), radius=16, fill=PRIMARY)
    d.text((590, 296), "Learn More", fill=WHITE, font=font_s)
    save(img, "feature-announcement.png")


def main() -> None:
    print("Generating lobby images...")
    draw_avatar()
    draw_icon_back()
    draw_icon_search()
    draw_notify()
    draw_announce()
    for k in ("lobby", "cloud", "member"):
        draw_tab(k)
    for k in ("copy", "custom", "download"):
        draw_bento(k)
    draw_news_item()
    draw_timer()
    draw_drag_handle()
    draw_icon_filter()
    draw_icon_chevron()
    draw_scheme_card()
    draw_feature_hero()
    print("Done.")


if __name__ == "__main__":
    main()
