#!/usr/bin/env python3
"""Generate GitHub Release notes from one CHANGELOG.md version section."""

from __future__ import annotations

import argparse
import re
from pathlib import Path


REQUIRED_SECTIONS = ("### 📦 升级说明",)


def extract_version_section(changelog: str, version: str) -> str:
    lines = changelog.splitlines()
    heading = re.compile(rf"^##\s+v?{re.escape(version)}\s*$", re.IGNORECASE)
    next_version = re.compile(r"^##\s+")

    start = None
    for index, line in enumerate(lines):
        if heading.match(line.strip()):
            start = index + 1
            break

    if start is None:
        raise ValueError(f"CHANGELOG.md 中未找到版本章节: {version}")

    end = len(lines)
    for index in range(start, len(lines)):
        if next_version.match(lines[index]):
            end = index
            break

    body = "\n".join(lines[start:end]).strip()
    if not body:
        raise ValueError(f"CHANGELOG.md 的版本章节为空: {version}")

    missing = [section for section in REQUIRED_SECTIONS if section not in body]
    if missing:
        raise ValueError(
            f"CHANGELOG.md 的 {version} 版本章节缺少发布必填内容: "
            + ", ".join(missing)
        )

    upgrade_match = re.search(
        r"^### 📦 升级说明\s*$\n(?P<content>.*?)(?=^###\s|\Z)",
        body,
        re.MULTILINE | re.DOTALL,
    )
    if not upgrade_match or not upgrade_match.group("content").strip():
        raise ValueError(f"CHANGELOG.md 的 {version} 升级说明不能为空")

    return body


def main() -> None:
    parser = argparse.ArgumentParser(description="从 CHANGELOG.md 生成 GitHub Release Notes")
    parser.add_argument("--version", required=True, help="SemVer，例如 1.1.0")
    parser.add_argument("--changelog", default="CHANGELOG.md", help="CHANGELOG 文件路径")
    parser.add_argument("--output", required=True, help="输出 Markdown 文件路径")
    args = parser.parse_args()

    section = extract_version_section(
        Path(args.changelog).read_text(encoding="utf-8"), args.version
    )
    notes = (
        f"> 本发布说明由 `CHANGELOG.md` 的 `{args.version}` 版本章节自动生成。\n\n"
        f"{section}\n"
    )
    output_path = Path(args.output)
    output_path.write_text(notes, encoding="utf-8")
    print(f"Release notes generated: {output_path}")


if __name__ == "__main__":
    main()
