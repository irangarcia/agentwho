#!/usr/bin/env python3

"""Encode full-frame PNG images as a browser-safe animated GIF."""

from pathlib import Path
import sys

from PIL import Image


def main() -> None:
    if len(sys.argv) < 5:
        raise SystemExit(
            "usage: encode_gif.py <output.gif> <delay-ms,...> <frame.png>..."
        )

    output = Path(sys.argv[1])
    durations = [int(value) for value in sys.argv[2].split(",")]
    frame_paths = [Path(value) for value in sys.argv[3:]]
    if len(durations) != len(frame_paths):
        raise SystemExit("the number of delays must match the number of frames")

    frames = [Image.open(path).convert("RGB") for path in frame_paths]
    try:
        frames[0].save(
            output,
            format="GIF",
            save_all=True,
            append_images=frames[1:],
            duration=durations,
            loop=0,
            disposal=2,
            optimize=False,
        )
    finally:
        for frame in frames:
            frame.close()


if __name__ == "__main__":
    main()
