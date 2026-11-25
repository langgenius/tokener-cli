import re
import sys
from pathlib import Path


tag, generated_path = sys.argv[1:]
changelog = Path("CHANGELOG.md").read_text()
sections = re.split(r"(?=^## \[)", changelog, flags=re.MULTILINE)
matching = [section.strip() for section in sections if section.startswith(f"## [{tag.removeprefix('v')}]\n")]
if len(matching) != 1:
    sys.exit(f"Expected one CHANGELOG.md section for {tag}; found {len(matching)}")
notes = matching[0]
if not re.search(r"^\* \S", notes, flags=re.MULTILINE):
    sys.exit(f"Empty release notes for {tag}")
if notes != Path(generated_path).read_text().strip():
    sys.exit(f"CHANGELOG.md does not match git-cliff output for {tag}")
print(notes)
