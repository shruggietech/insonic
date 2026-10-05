"""Run the immutable approved helper with a resolved BrandBuilder library."""

import argparse
import runpy
import sys
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--templates', type=Path, help='Installed BrandBuilder templates directory')
    args = parser.parse_args()
    here = Path(__file__).resolve().parent
    library = args.templates
    if library is None:
        library = next((candidate for ancestor in here.parents
                        for candidate in (ancestor / 'skill' / 'templates', ancestor / 'templates')
                        if (candidate / 'glyphkit.py').is_file()), None)
    if library is None or not (library / 'glyphkit.py').is_file():
        parser.error('Cannot find glyphkit.py. Pass --templates with the installed BrandBuilder templates directory.')
    sys.path.insert(0, str(library.resolve()))
    runpy.run_path(str(here / 'mk_paths.py'), run_name='__main__')
    return 0


if __name__ == '__main__':
    sys.exit(main())
