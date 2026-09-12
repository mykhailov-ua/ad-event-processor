<?php

if (!defined('ABSPATH')) {
    exit;
}

final class AED_Tracker_Shortcode {
    public static function init(): void {
        add_shortcode('aed_click_link', [self::class, 'render_click_link']);
        add_action('wp_enqueue_scripts', [self::class, 'enqueue_track_script']);
    }

    public static function enqueue_track_script(): void {
        $settings = AED_Tracker_Settings::get();
        $tracker = rtrim((string) ($settings['tracker_url'] ?? ''), '/');
        if ($tracker === '') {
            return;
        }
        wp_enqueue_script(
            'aed-track',
            $tracker . '/static/track.js',
            [],
            AED_TRACKER_VERSION,
            true
        );
    }

    public static function render_click_link($atts): string {
        $atts = shortcode_atts([
            'label' => 'Continue',
            'campaign_id' => '',
            'class' => 'aed-click-link',
        ], $atts, 'aed_click_link');

        $settings = AED_Tracker_Settings::get();
        $campaign_id = trim((string) $atts['campaign_id']);
        if ($campaign_id === '') {
            $campaign_id = trim((string) ($settings['default_campaign_id'] ?? ''));
        }
        if ($campaign_id === '') {
            return '<!-- aed_click_link: campaign_id required -->';
        }

        $click_url = self::mint_click_url($settings, $campaign_id);
        if ($click_url === '') {
            return '<!-- aed_click_link: click mint failed -->';
        }

        $label = esc_html((string) $atts['label']);
        $class = esc_attr((string) $atts['class']);
        $href = esc_url($click_url);

        return '<a class="' . $class . '" href="' . $href . '" rel="nofollow noopener">' . $label . '</a>';
    }

    private static function mint_click_url(array $settings, string $campaign_id): string {
        $base = rtrim((string) ($settings['control_plane_url'] ?? ''), '/');
        $token = trim((string) ($settings['api_token'] ?? ''));
        if ($base === '' || $token === '') {
            return '';
        }

        $response = wp_remote_post(
            $base . '/api/v1/tracker/clicks',
            [
                'timeout' => 15,
                'headers' => [
                    'Authorization' => 'Bearer ' . $token,
                    'Content-Type' => 'application/json',
                ],
                'body' => wp_json_encode([
                    'campaign_id' => $campaign_id,
                ]),
            ]
        );

        if (is_wp_error($response)) {
            return '';
        }
        $code = (int) wp_remote_retrieve_response_code($response);
        if ($code < 200 || $code >= 300) {
            return '';
        }
        $body = json_decode((string) wp_remote_retrieve_body($response), true);
        if (!is_array($body) || empty($body['click_url'])) {
            return '';
        }
        return (string) $body['click_url'];
    }
}
