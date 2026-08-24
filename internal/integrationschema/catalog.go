package integrationschema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/platformconfig"
)

type TemplateCatalogEntry struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Version  int    `json:"version"`
	Category string `json:"category"`
	Kind     Kind   `json:"kind"`
}

var BundledIntegrationTemplateCatalog = []TemplateCatalogEntry{
	{Name: "traffic_propellerads", File: "traffic_propellerads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_exoclick", File: "traffic_exoclick.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_facebook", File: "traffic_facebook.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_popads", File: "traffic_popads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_trafficstars", File: "traffic_trafficstars.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adsterra", File: "traffic_adsterra.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_clickadu", File: "traffic_clickadu.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_mgid", File: "traffic_mgid.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_hilltopads", File: "traffic_hilltopads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_richads", File: "traffic_richads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_google_ads", File: "traffic_google_ads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_microsoft_ads", File: "traffic_microsoft_ads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_tiktok", File: "traffic_tiktok.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_taboola", File: "traffic_taboola.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_outbrain", File: "traffic_outbrain.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_pushground", File: "traffic_pushground.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_galaksion", File: "traffic_galaksion.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_revcontent", File: "traffic_revcontent.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_juicyads", File: "traffic_juicyads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_evadav", File: "traffic_evadav.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_zeropark", File: "traffic_zeropark.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_snapchat", File: "traffic_snapchat.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_linkedin", File: "traffic_linkedin.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_instagram", File: "traffic_instagram.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_threads", File: "traffic_threads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_pinterest", File: "traffic_pinterest.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_x_ads", File: "traffic_x_ads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_reddit", File: "traffic_reddit.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_youtube", File: "traffic_youtube.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adcash", File: "traffic_adcash.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_admaven", File: "traffic_admaven.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_mondiad", File: "traffic_mondiad.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_trafficjunky", File: "traffic_trafficjunky.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_pushhouse", File: "traffic_pushhouse.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_clickadilla", File: "traffic_clickadilla.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_ezmob", File: "traffic_ezmob.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_twinred", File: "traffic_twinred.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_trafficfactory", File: "traffic_trafficfactory.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_yandex_direct", File: "traffic_yandex_direct.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_rollerads", File: "traffic_rollerads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_noviclick", File: "traffic_noviclick.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_popcash", File: "traffic_popcash.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adoperator", File: "traffic_adoperator.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_traffic_nomads", File: "traffic_traffic_nomads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_admixer", File: "traffic_admixer.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adnium", File: "traffic_adnium.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adsupply", File: "traffic_adsupply.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_bitmedia", File: "traffic_bitmedia.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_mediago", File: "traffic_mediago.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_rtx_platform", File: "traffic_rtx_platform.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_sourceknowledge", File: "traffic_sourceknowledge.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_reacheffect", File: "traffic_reacheffect.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_kadam", File: "traffic_kadam.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_plugrush", File: "traffic_plugrush.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_runative", File: "traffic_runative.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adskeeper", File: "traffic_adskeeper.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adxad", File: "traffic_adxad.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_yeesshh", File: "traffic_yeesshh.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adstyle", File: "traffic_adstyle.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_advedro", File: "traffic_advedro.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_geospot_media", File: "traffic_geospot_media.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_rtb_panda", File: "traffic_rtb_panda.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_widget_media", File: "traffic_widget_media.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_onclicka", File: "traffic_onclicka.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_bidvertiser", File: "traffic_bidvertiser.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_dao_ad", File: "traffic_dao_ad.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_active_revenue", File: "traffic_active_revenue.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_clickaine", File: "traffic_clickaine.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_targeleon", File: "traffic_targeleon.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_smartyads", File: "traffic_smartyads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_traffic_shark", File: "traffic_traffic_shark.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_adkernel", File: "traffic_adkernel.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_reporo", File: "traffic_reporo.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_clickflow", File: "traffic_clickflow.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_epom", File: "traffic_epom.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_mybid", File: "traffic_mybid.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_pushame", File: "traffic_pushame.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_vimmy", File: "traffic_vimmy.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_traffic_force", File: "traffic_traffic_force.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_admeking", File: "traffic_admeking.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_clickstar", File: "traffic_clickstar.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_selfadvertiser", File: "traffic_selfadvertiser.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "affiliate_everad", File: "affiliate_everad_postback.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindOutboundPostback},
	{Name: "affiliate_everad_status", File: "affiliate_everad_status.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindStatusMapping},
	{Name: "affiliate_leadbit", File: "affiliate_leadbit_postback.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindOutboundPostback},
	{Name: "affiliate_leadbit_status", File: "affiliate_leadbit_status.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindStatusMapping},
	{Name: "affiliate_adcombo", File: "affiliate_adcombo_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_adcombo_status", File: "affiliate_adcombo_status.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindStatusMapping},
	{Name: "affiliate_lospollos", File: "affiliate_lospollos_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_terraleads", File: "affiliate_terraleads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_dr_cash", File: "affiliate_dr_cash_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_cpamatica", File: "affiliate_cpamatica_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_mobidea", File: "affiliate_mobidea_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_mylead", File: "affiliate_mylead_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_maxbounty", File: "affiliate_maxbounty_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_clickdealer", File: "affiliate_clickdealer_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_lemonads", File: "affiliate_lemonads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_zeydoo", File: "affiliate_zeydoo_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_crakrevenue", File: "affiliate_crakrevenue_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_advidi", File: "affiliate_advidi_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_alfaleads", File: "affiliate_alfaleads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_leadxchange", File: "affiliate_leadxchange_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_olimob", File: "affiliate_olimob_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_yepads", File: "affiliate_yepads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_creativeclicks", File: "affiliate_creativeclicks_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_cpahub", File: "affiliate_cpahub_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_traffic_company", File: "affiliate_traffic_company_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_digistore24", File: "affiliate_digistore24_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_m4trix", File: "affiliate_m4trix_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_nova", File: "affiliate_nova_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_pwn_games", File: "affiliate_pwn_games_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_maxweb", File: "affiliate_maxweb_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_bluepartner", File: "affiliate_bluepartner_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_franktrax", File: "affiliate_franktrax_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_gold_lead", File: "affiliate_gold_lead_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_hugeoffers", File: "affiliate_hugeoffers_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_media500", File: "affiliate_media500_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_mobipium", File: "affiliate_mobipium_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_direct_affiliate", File: "affiliate_direct_affiliate_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_buygoods", File: "affiliate_buygoods_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_kelkoo", File: "affiliate_kelkoo_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_adultforce", File: "affiliate_adultforce_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_gamesvid", File: "affiliate_gamesvid_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_datify", File: "affiliate_datify_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_traforama", File: "affiliate_traforama_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_ytz", File: "affiliate_ytz_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_big_bang_ads", File: "affiliate_big_bang_ads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_clickbank", File: "affiliate_clickbank_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_tonic_affiliate", File: "affiliate_tonic_affiliate_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_shakes", File: "affiliate_shakes_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_aff1", File: "affiliate_aff1_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_royal_partners", File: "affiliate_royal_partners_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_pinup_partners", File: "affiliate_pinup_partners_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_1win_partners", File: "affiliate_1win_partners_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_mostbet_partners", File: "affiliate_mostbet_partners_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_ogads", File: "affiliate_ogads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_cpalead", File: "affiliate_cpalead_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_aivix", File: "affiliate_aivix_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_peerfly", File: "affiliate_peerfly_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_kma_biz", File: "affiliate_kma_biz_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_gasmobi", File: "affiliate_gasmobi_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_performcb", File: "affiliate_performcb_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_adwork_media", File: "affiliate_adwork_media_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_golden_goose", File: "affiliate_golden_goose_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_leadgid", File: "affiliate_leadgid_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_monetizer", File: "affiliate_monetizer_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_nutryst", File: "affiliate_nutryst_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_toro_advertising", File: "affiliate_toro_advertising_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_smartadv", File: "affiliate_smartadv_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_arrow_media", File: "affiliate_arrow_media_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_revenuelab", File: "affiliate_revenuelab_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_convertize", File: "affiliate_convertize_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_hugoads", File: "affiliate_hugoads_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_instal", File: "affiliate_instal_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_cpagrip", File: "affiliate_cpagrip_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_wiget", File: "affiliate_wiget_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_jm_solution", File: "affiliate_jm_solution_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_adskill", File: "affiliate_adskill_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
	{Name: "affiliate_gotzha", File: "affiliate_gotzha_receive.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindAffiliateReceivePostback},
}

var affiliateStatusPairs = map[string]string{
	"affiliate_everad":  "affiliate_everad_status",
	"affiliate_leadbit": "affiliate_leadbit_status",
	"affiliate_adcombo": "affiliate_adcombo_status",
}

func AffiliateStatusTemplateName(network string) (string, bool) {
	name, ok := affiliateStatusPairs[strings.TrimSpace(network)]
	return name, ok
}

func SchemaRootDir() string {
	if root := config.InstallRootFromEnv(); root != "" {
		return filepath.Join(root, "deploy", "schemas")
	}
	candidates := []string{
		filepath.Join("deploy", "schemas"),
		filepath.Join("..", "..", "deploy", "schemas"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

func LoadBundledTemplate(entry TemplateCatalogEntry) (raw []byte, kind Kind, schema any, err error) {
	raw, err = os.ReadFile(filepath.Join(SchemaRootDir(), entry.File))
	if err != nil {
		return nil, "", nil, fmt.Errorf("read %s: %w", entry.File, err)
	}
	kind, parsed, err := ParseDocument(raw)
	if err != nil {
		return nil, "", nil, err
	}
	if entry.Kind != "" && kind != entry.Kind {
		return nil, "", nil, fmt.Errorf("template %s: expected kind %s, got %s", entry.Name, entry.Kind, kind)
	}
	return raw, kind, parsed, nil
}

func FindCatalogEntry(name string) (TemplateCatalogEntry, bool) {
	name = strings.TrimSpace(name)
	for _, e := range BundledIntegrationTemplateCatalog {
		if e.Name == name {
			return e, true
		}
	}
	return TemplateCatalogEntry{}, false
}

func BuildInboundTrackingURL(trackingDomain string, s *InboundTokensSchema) string {
	host := platformconfig.ResolveHost(trackingDomain)
	if host == "" {
		host = "track.example.com"
	}
	var parts []string
	parts = append(parts, "campaign_id={campaign_id}")
	seen := map[string]struct{}{"campaign_id": {}}
	for _, t := range s.Tokens {
		key := strings.TrimSpace(t.QueryKey)
		if key == "" || key == "campaign_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"={"+key+"}")
	}
	for _, m := range s.Macros {
		key := strings.TrimSpace(m.Key)
		if key == "" || key == "campaign_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"={"+key+"}")
	}
	return fmt.Sprintf("https://%s/click?%s", host, strings.Join(parts, "&"))
}
