<?php

if (!defined('ABSPATH')) {
    exit;
}

final class AED_Tracker_Settings {
    public const OPTION_KEY = 'aed_tracker_settings';

    public static function init(): void {
        add_action('admin_menu', [self::class, 'register_menu']);
        add_action('admin_init', [self::class, 'register_settings']);
    }

    public static function register_menu(): void {
        add_options_page(
            'Ad Event Tracker',
            'Ad Event Tracker',
            'manage_options',
            'aed-tracker',
            [self::class, 'render_page']
        );
    }

    public static function register_settings(): void {
        register_setting(self::OPTION_KEY, self::OPTION_KEY, [
            'type' => 'array',
            'sanitize_callback' => [self::class, 'sanitize'],
            'default' => [],
        ]);
    }

    public static function get(): array {
        $stored = get_option(self::OPTION_KEY, []);
        return is_array($stored) ? $stored : [];
    }

    public static function sanitize($input): array {
        $input = is_array($input) ? $input : [];
        return [
            'control_plane_url' => esc_url_raw(trim((string) ($input['control_plane_url'] ?? ''))),
            'tracker_url' => esc_url_raw(trim((string) ($input['tracker_url'] ?? ''))),
            'api_token' => sanitize_text_field((string) ($input['api_token'] ?? '')),
            'default_campaign_id' => sanitize_text_field((string) ($input['default_campaign_id'] ?? '')),
        ];
    }

    public static function render_page(): void {
        if (!current_user_can('manage_options')) {
            return;
        }
        $settings = self::get();
        ?>
        <div class="wrap">
            <h1>Ad Event Tracker</h1>
            <p>Configure control plane API access for programmatic clicks and browser track.js.</p>
            <form method="post" action="options.php">
                <?php settings_fields(self::OPTION_KEY); ?>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row"><label for="aed_control_plane_url">Control plane URL</label></th>
                        <td><input name="<?php echo esc_attr(self::OPTION_KEY); ?>[control_plane_url]" id="aed_control_plane_url" type="url" class="regular-text" value="<?php echo esc_attr($settings['control_plane_url'] ?? ''); ?>" placeholder="https://control.example.com" /></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="aed_tracker_url">Tracker URL</label></th>
                        <td><input name="<?php echo esc_attr(self::OPTION_KEY); ?>[tracker_url]" id="aed_tracker_url" type="url" class="regular-text" value="<?php echo esc_attr($settings['tracker_url'] ?? ''); ?>" placeholder="https://trk.example.com" /></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="aed_api_token">API Bearer token</label></th>
                        <td><input name="<?php echo esc_attr(self::OPTION_KEY); ?>[api_token]" id="aed_api_token" type="password" class="regular-text" value="<?php echo esc_attr($settings['api_token'] ?? ''); ?>" autocomplete="off" /></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="aed_campaign_id">Default campaign ID</label></th>
                        <td><input name="<?php echo esc_attr(self::OPTION_KEY); ?>[default_campaign_id]" id="aed_campaign_id" type="text" class="regular-text" value="<?php echo esc_attr($settings['default_campaign_id'] ?? ''); ?>" /></td>
                    </tr>
                </table>
                <?php submit_button(); ?>
            </form>
            <p>Shortcode: <code>[aed_click_link label="Buy now" campaign_id="UUID"]</code></p>
        </div>
        <?php
    }
}
