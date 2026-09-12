<?php
/**
 * Plugin Name: Ad Event Tracker
 * Description: Shortcode and settings for ad-event-processor programmatic click API and track.js.
 * Version: 1.0.0
 * Requires at least: 6.0
 * Requires PHP: 7.4
 * Author: ad-event-processor
 * License: GPLv2 or later
 */

if (!defined('ABSPATH')) {
    exit;
}

define('AED_TRACKER_VERSION', '1.0.0');
define('AED_TRACKER_PLUGIN_FILE', __FILE__);
define('AED_TRACKER_PLUGIN_DIR', plugin_dir_path(__FILE__));

require_once AED_TRACKER_PLUGIN_DIR . 'includes/class-settings.php';
require_once AED_TRACKER_PLUGIN_DIR . 'includes/class-shortcode.php';

final class AED_Tracker_Plugin {
    public static function init(): void {
        AED_Tracker_Settings::init();
        AED_Tracker_Shortcode::init();
    }
}

add_action('plugins_loaded', ['AED_Tracker_Plugin', 'init']);
