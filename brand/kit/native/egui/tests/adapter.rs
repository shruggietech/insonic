use egui_kittest::{Harness, kittest::Queryable as _};
use shruggie_insonic_egui::components::{app_frame, apply_capabilities, apply_style, button, card, checkbox, dialog, empty_state, field, icon_button, list_row, menu, split_pane, status_badge, tabs, toast, toolbar, ButtonIntent};
use shruggie_insonic_egui::{Density, Insets, PointerPrecision, RuntimeCapabilities, ThemeMode};
use shruggie_insonic_egui::tokens::{logical_points, Tokens, UnitTransformError};

fn contrast(left: egui::Color32, right: egui::Color32) -> f32 {
    fn linear(channel: u8) -> f32 {
        let value = channel as f32 / 255.0;
        if value <= 0.04045 { value / 12.92 } else { ((value + 0.055) / 1.055).powf(2.4) }
    }
    fn luminance(color: egui::Color32) -> f32 {
        0.2126 * linear(color.r()) + 0.7152 * linear(color.g()) + 0.0722 * linear(color.b())
    }
    let a = luminance(left);
    let b = luminance(right);
    (a.max(b) + 0.05) / (a.min(b) + 0.05)
}

fn capabilities(text_scale: f32) -> RuntimeCapabilities {
    RuntimeCapabilities {
        viewport_points: egui::vec2(1280.0, 720.0),
        safe_area: Insets { top: 0.0, right: 0.0, bottom: 0.0, left: 0.0 },
        window_class: "expanded",
        pointer_precision: PointerPrecision::Fine,
        hover: true,
        hardware_keyboard: true,
        touch: false,
        text_scale,
        reduced_motion: false,
        forced_colors: false,
        ime_obstruction: None,
        titlebar_regions: Vec::new(),
        theme: ThemeMode::System,
        density: Density::Compact,
    }
}

#[test]
fn button_input_and_accessible_name_are_rendered() {
    let mut harness = Harness::new_ui_state(|ui, count: &mut usize| {
        if button(ui, "Save", ButtonIntent::Primary, false).clicked() { *count += 1; }
        icon_button(ui, "Close", "x", false);
    }, 0usize);
    harness.get_by_label("Close");
    harness.get_by_label("Save").click();
    harness.run();
    assert_eq!(*harness.state(), 1);
}

#[test]
fn selection_and_error_state_are_rendered() {
    #[derive(Default)] struct State { selected: usize, value: String }
    let mut harness = Harness::new_ui_state(|ui, state: &mut State| {
        tabs(ui, &mut state.selected, &["One", "Two"]);
        field(ui, "Name", &mut state.value, Some("Required"));
    }, State::default());
    harness.get_by_label("Two").click();
    harness.run();
    assert_eq!(harness.state().selected, 1);
    harness.get_by_label("Name");
    harness.get_by_label("Required");
}

#[test]
fn complete_recipe_grammar_renders_in_egui() {
    #[derive(Default)] struct State { selected: usize, value: String, checked: bool, row: bool, dialog_open: bool }
    let mut harness = Harness::new_ui_state(|ui, state: &mut State| {
        state.dialog_open = true;
        let runtime = capabilities(1.0);
        app_frame(ui, &runtime, |ui| {
            toolbar(ui, false, |ui| {
                button(ui, "Primary", ButtonIntent::Primary, false);
                icon_button(ui, "Settings", "*", false);
            });
            tabs(ui, &mut state.selected, &["Alpha", "Beta"]);
            menu(ui, "Actions", |ui| ui.button("Run"));
            field(ui, "Project", &mut state.value, None);
            checkbox(ui, &mut state.checked, "Enabled");
            list_row(ui, &mut state.row, "Row");
            split_pane(ui, |ui| ui.label("Primary pane"), |ui| ui.label("Secondary pane"));
            toast(ui, "Saved", false);
            status_badge(ui, "Ready", false);
            card(ui, |ui| ui.label("Card content"));
            empty_state(ui, "Nothing here", "Create the first item");
        }).unwrap();
        let context = ui.ctx().clone();
        dialog(&context, &mut state.dialog_open, "Confirm", |ui| ui.label("Dialog content"));
    }, State::default());
    harness.run();
    for label in ["Primary", "Settings", "Alpha", "Actions", "Project", "Enabled", "Row", "Saved", "Ready", "Nothing here", "Confirm"] {
        harness.get_by_label(label);
    }
}

#[test]
fn density_and_pixels_per_point_remain_independent() {
    let comfortable = Tokens::for_theme(ThemeMode::Dark, Density::Comfortable);
    let compact = Tokens::for_theme(ThemeMode::Dark, Density::Compact);
    assert!(compact.spacing_component < comfortable.spacing_component);
    assert_eq!(compact.target_minimum, comfortable.target_minimum);
    let mut harness = Harness::new_ui(|ui| { ui.label("Scaled"); });
    harness.set_pixels_per_point(2.0);
    harness.run();
    assert_eq!(harness.ctx.pixels_per_point(), 2.0);
}

#[test]
fn fine_pointer_controls_and_rows_use_compact_visual_density() {
    let context = egui::Context::default();
    let mut runtime = capabilities(1.0);
    runtime.theme = ThemeMode::Dark;
    runtime.density = Density::Comfortable;
    apply_capabilities(&context, &runtime).unwrap();
    let comfortable = context.style_of(egui::Theme::Dark);
    assert!(comfortable.spacing.interact_size.y <= 28.0);
    assert!(comfortable.spacing.button_padding.y <= 4.0);

    runtime.density = Density::Compact;
    apply_capabilities(&context, &runtime).unwrap();
    let compact = context.style_of(egui::Theme::Dark);
    assert!(compact.spacing.interact_size.y <= 24.0);
    assert!(compact.spacing.interact_size.y < comfortable.spacing.interact_size.y);
    assert!(compact.spacing.item_spacing.y <= 2.0);
    assert!(compact.spacing.button_padding.y < comfortable.spacing.button_padding.y);
}

#[test]
fn consecutive_log_rows_render_with_near_contiguous_spacing() {
    let mut harness = Harness::new_ui_state(|ui, gap: &mut f32| {
        apply_style(ui.ctx(), ThemeMode::Dark, Density::Compact);
        let first = ui.label("2026-09-22T00:54:55Z INFO first event");
        let second = ui.label("2026-09-22T00:54:56Z INFO second event");
        *gap = second.rect.top() - first.rect.bottom();
    }, 0.0f32);
    harness.run();
    assert!(*harness.state() <= 2.0, "log row gap was {}", harness.state());
}

#[test]
fn conservative_inputs_retain_the_governed_target() {
    let context = egui::Context::default();
    let governed = Tokens::for_theme(ThemeMode::Dark, Density::Compact).target_minimum;
    for precision in [PointerPrecision::None, PointerPrecision::Coarse, PointerPrecision::Mixed] {
        let mut runtime = capabilities(1.0);
        runtime.theme = ThemeMode::Dark;
        runtime.pointer_precision = precision;
        apply_capabilities(&context, &runtime).unwrap();
        assert!(context.style_of(egui::Theme::Dark).spacing.interact_size.y >= governed);
    }
    let mut touch = capabilities(1.0);
    touch.theme = ThemeMode::Dark;
    touch.touch = true;
    apply_capabilities(&context, &touch).unwrap();
    assert!(context.style_of(egui::Theme::Dark).spacing.interact_size.y >= governed);
}

#[test]
fn text_scaling_can_grow_a_compact_control() {
    let context = egui::Context::default();
    let mut normal = capabilities(1.0);
    normal.theme = ThemeMode::Dark;
    apply_capabilities(&context, &normal).unwrap();
    let normal_height = context.style_of(egui::Theme::Dark).spacing.interact_size.y;
    let mut scaled = capabilities(3.0);
    scaled.theme = ThemeMode::Dark;
    apply_capabilities(&context, &scaled).unwrap();
    assert!(context.style_of(egui::Theme::Dark).spacing.interact_size.y > normal_height);
}

#[test]
fn invalid_units_and_runtime_scaling_fail_closed() {
    assert_eq!(logical_points(f32::NAN), Err(UnitTransformError::NonFinite));
    assert_eq!(logical_points(0.0), Err(UnitTransformError::NonPositive));
    assert_eq!(logical_points(2.0).unwrap(), 2.0);
    let mut capabilities = capabilities(1.5);
    let context = egui::Context::default();
    apply_capabilities(&context, &capabilities).unwrap();
    assert_eq!(context.options(|options| options.theme_preference), egui::ThemePreference::System);
    capabilities.text_scale = 0.0;
    assert!(apply_capabilities(&context, &capabilities).is_err());
}

#[test]
fn generated_style_applies_focus_selection_tokens() {
    let mut harness = Harness::new_ui(|ui| { apply_style(ui.ctx(), ThemeMode::Light, Density::Comfortable); let _ = ui.button("Focus target"); });
    harness.run();
    let expected = Tokens::for_theme(ThemeMode::Light, Density::Comfortable).action;
    assert_eq!(harness.ctx.style_of(egui::Theme::Light).visuals.selection.bg_fill, expected);
}

#[test]
fn native_status_and_control_states_remain_readable_in_both_themes() {
    for theme in [ThemeMode::Light, ThemeMode::Dark] {
        let mut harness = Harness::new_ui(move |ui| {
            apply_style(ui.ctx(), theme, Density::Comfortable);
            ui.strong("Capture Unavailable");
            button(ui, "Available", ButtonIntent::Primary, false);
            button(ui, "Unavailable", ButtonIntent::Primary, true);
        });
        harness.run();
        harness.get_by_label("Capture Unavailable");
        harness.get_by_label("Unavailable");
        harness.get_by_label("Available").hover();
        harness.run();
        harness.get_by_label("Available").focus();
        harness.run();
        let actual_theme = match theme { ThemeMode::Light => egui::Theme::Light, _ => egui::Theme::Dark };
        let style = harness.ctx.style_of(actual_theme);
        let tokens = Tokens::for_theme(theme, Density::Comfortable);
        let disabled_text = harness.output().shapes.iter().filter_map(|clipped| match &clipped.shape {
            egui::Shape::Text(text) if text.galley.text() == "Unavailable" => Some(text),
            _ => None,
        }).collect::<Vec<_>>();
        assert!(!disabled_text.is_empty());
        for text in disabled_text {
            assert_eq!(text.opacity_factor, 1.0);
            let colors = text.galley.rows.iter().flat_map(|row| row.row.visuals.mesh.vertices.iter().map(|vertex| vertex.color)).collect::<Vec<_>>();
            assert!(!colors.is_empty());
            assert!(colors.iter().all(|color| color.a() == 255 && contrast(*color, tokens.card) >= 4.5));
        }
        assert_eq!(style.visuals.strong_text_color(), tokens.text_primary);
        assert!(contrast(style.visuals.strong_text_color(), tokens.card) >= 4.5);
        assert!(contrast(tokens.text_muted, tokens.card) >= 4.5);
        assert!(contrast(tokens.on_action, tokens.action) >= 4.5);
        assert_eq!(style.visuals.widgets.hovered.fg_stroke.color, tokens.text_primary);
        assert_eq!(style.visuals.selection.stroke.color, tokens.focus);
    }
}
