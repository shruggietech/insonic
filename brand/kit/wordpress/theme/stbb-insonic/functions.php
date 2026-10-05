<?php
/** Generated content-only stylesheet hooks. */
add_action( 'after_setup_theme', function () { add_theme_support( 'editor-styles' ); add_editor_style( 'assets/css/stbb-content.css' ); } );
add_action( 'init', function () { register_block_style( 'core/group', array( 'name' => 'stbb-insonic-card', 'label' => 'Brand card' ) ); } );
add_action( 'wp_enqueue_scripts', function () { wp_enqueue_style( 'stbb-insonic-content', get_theme_file_uri( 'assets/css/stbb-content.css' ), array(), '2218715c503cde2a75c617ed30febeb438a09e860795c8e06c1716a96a173728' ); } );
