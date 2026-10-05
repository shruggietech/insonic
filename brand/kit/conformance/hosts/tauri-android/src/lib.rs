#[derive(Clone, Copy, Debug)]
pub struct Rect { pub x: f32, pub y: f32, pub width: f32, pub height: f32 }

#[derive(Clone, Copy, Debug)]
pub struct Insets { pub top: f32, pub right: f32, pub bottom: f32, pub left: f32 }

#[derive(Clone, Copy, Debug)]
pub struct Envelope {
    pub viewport_width: f32,
    pub viewport_height: f32,
    pub safe: Insets,
    pub ime_block_end: f32,
    pub safe_area_owners: u8,
    pub ime_owners: u8,
    pub required_control: Rect,
}

pub fn diagnostics(envelope: Envelope) -> Vec<&'static str> {
    let mut result = Vec::new();
    if envelope.safe_area_owners != 1 || envelope.ime_owners != 1 {
        result.push("host.duplicate-owner");
    }
    let safe_count = envelope.safe_area_owners.max(1) as f32;
    let ime_count = envelope.ime_owners.max(1) as f32;
    let left = envelope.safe.left * safe_count;
    let right = envelope.viewport_width - envelope.safe.right * safe_count;
    let top = envelope.safe.top * safe_count;
    let bottom = envelope.viewport_height - envelope.safe.bottom * safe_count - envelope.ime_block_end * ime_count;
    let control = envelope.required_control;
    let contained = control.x >= left && control.y >= top && control.x + control.width <= right && control.y + control.height <= bottom;
    if !contained { result.push("host.control-obstructed"); }
    result
}
