import re
import sys
from pathlib import Path


tag, generated_path = sys.argv[1:]


def release_section(path):
    sections = re.split(r"(?=^## \[)", Path(path).read_text(), flags=re.MULTILINE)
    matching = [section.strip() for section in sections if section.startswith(f"## [{tag.removeprefix('v')}]\n")]
    if len(matching) != 1:
        sys.exit(f"Expected one {path} section for {tag}; found {len(matching)}")
    if not re.search(r"^\* \S", matching[0], flags=re.MULTILINE):
        sys.exit(f"Empty release notes for {tag} in {path}")
    return matching[0]


notes = release_section("CHANGELOG.md")
if notes != release_section(generated_path):
    sys.exit(f"CHANGELOG.md does not match git-cliff output for {tag}")
print(notes)
