#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""One disposable console-only decoder preflight; never starts product code."""
import importlib.util
import json
from pathlib import Path

from windows_scheduler_diagnostic import Scheduler

ROOT=Path(__file__).resolve().parents[1]


def run(tools=None):
    if tools is None:
        spec=importlib.util.spec_from_file_location('scheduler_preflight_tools',ROOT/'scripts/diagnose-desktop-startup.py')
        tools=importlib.util.module_from_spec(spec);spec.loader.exec_module(tools)
    tools.pipeline.m.disposable_runner()
    evidence=ROOT/'build/scheduler-preflight/evidence'
    tools.save(evidence,'preflight.json',{'available':False,'feature_qualification':False,'gui_launches':0,
        'reason':'observer-initialization-not-completed','raw_trace_retained':False})
    try:created=Scheduler(ROOT,tools.capture,tools.pipeline.m.disposable_runner)
    except Exception:
        tools.save(evidence,'cleanup.json',{'confirmed':False,'session_started':False,'raw_directory_removed':None,
            'feature_qualification':False,'gui_launches':0})
        return 78
    with created as observer:
        try:
            observer.prepare()
            result={'available':True,'preflight':observer.preflight,'feature_qualification':False,'gui_launches':0,
                    'raw_trace_retained':False,'cleanup_confirmed':observer.cleanup_confirmed}
        except Exception:
            result={**observer.failure(),'gui_launches':0}
            observer.cancel();result['cleanup_confirmed']=observer.cleanup_confirmed
        tools.save(evidence,'preflight.json',result)
    tools.save(evidence,'cleanup.json',{'confirmed':observer.cleanup_confirmed,'raw_directory_removed':not observer.private.exists(),
        'feature_qualification':False,'gui_launches':0})
    return 0 if result['available'] and observer.cleanup_confirmed else 78


if __name__=='__main__':raise SystemExit(run())
