use insonic_tauri_android_conformance::{diagnostics, Envelope, Insets, Rect};

#[test]
fn known_bad_double_consumption_fails() {
    let failures = diagnostics(Envelope {
        viewport_width: 412.0, viewport_height: 915.0,
        safe: Insets { top: 38.0, right: 0.0, bottom: 24.0, left: 0.0 },
        ime_block_end: 0.0, safe_area_owners: 2, ime_owners: 1,
        required_control: Rect { x: 16.0, y: 850.0, width: 56.0, height: 56.0 },
    });
    assert!(failures.contains(&"host.duplicate-owner"));
    assert!(failures.contains(&"host.control-obstructed"));
}

#[test]
fn corrected_orientation_and_ime_states_pass() {
    let states = [
        Envelope { viewport_width: 412.0, viewport_height: 915.0, safe: Insets { top: 38.0, right: 0.0, bottom: 24.0, left: 0.0 }, ime_block_end: 0.0, safe_area_owners: 1, ime_owners: 1, required_control: Rect { x: 16.0, y: 812.0, width: 56.0, height: 56.0 } },
        Envelope { viewport_width: 915.0, viewport_height: 412.0, safe: Insets { top: 0.0, right: 24.0, bottom: 20.0, left: 38.0 }, ime_block_end: 160.0, safe_area_owners: 1, ime_owners: 1, required_control: Rect { x: 54.0, y: 148.0, width: 300.0, height: 44.0 } },
        Envelope { viewport_width: 430.0, viewport_height: 780.0, safe: Insets { top: 34.0, right: 0.0, bottom: 22.0, left: 0.0 }, ime_block_end: 0.0, safe_area_owners: 1, ime_owners: 1, required_control: Rect { x: 16.0, y: 690.0, width: 56.0, height: 56.0 } },
    ];
    for state in states { assert!(diagnostics(state).is_empty()); }
}
