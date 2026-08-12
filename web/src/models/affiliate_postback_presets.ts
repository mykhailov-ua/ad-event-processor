/**
 * Affiliate-network postback URL presets for outbound webhook config (§11.4).
 * Static catalog in the admin bundle — BidShard macros only ({click_id}, {payout}, …).
 */

export type AffiliatePostbackPreset = {
  id: string;
  name: string;
  /**
   * Outbound URL template (BidShard → partner) using BidShard macros.
   * Operators replace the host / path with the network-provided postback endpoint when needed.
   */
  url_template: string;
  /** Network-side name for click / subid (docs hint). */
  network_click_token: string;
  /** Network-side name for payout / amount. */
  network_payout_token: string;
  notes?: string;
};

function preset(
  id: string,
  name: string,
  query: string,
  clickToken: string,
  payoutToken: string,
  notes?: string,
): AffiliatePostbackPreset {
  return {
    id,
    name,
    url_template: `https://REPLACE_HOST/postback?${query}`,
    network_click_token: clickToken,
    network_payout_token: payoutToken,
    notes,
  };
}

/**
 * Canonical 36-network catalog (MaxBounty → Custom).
 */
export const AFFILIATE_POSTBACK_PRESETS: AffiliatePostbackPreset[] = [
  preset('maxbounty', 'MaxBounty', 'cid={click_id}&payout={payout}&txid={tx_id}', '{subid}', '{payout}',
    'Map MaxBounty {subid} → BidShard {click_id} on inbound; outbound uses BidShard macros.'),
  preset('clickdealer', 'ClickDealer', 'clickid={click_id}&payout={payout}&txid={tx_id}', '#s1#', '#price#'),
  preset('admitad', 'Admitad', 'click_id={click_id}&amount={payout}&order_id={tx_id}', '{{{subid}}}', '{{{price}}}'),
  preset('awin', 'Awin', 'clickRef={click_id}&amount={payout}&orderRef={tx_id}', '!!clickref!!', '!!amount!!'),
  preset('cj', 'CJ Affiliate', 'CID={click_id}&AMOUNT={payout}&OID={tx_id}', 'SID', 'AMOUNT'),
  preset('shareasale', 'ShareASale', 'tracking={click_id}&amount={payout}&transtype=sale&ordernumber={tx_id}', 'sscid', 'amount'),
  preset('rakuten', 'Rakuten Advertising', 'u1={click_id}&amt={payout}&ord={tx_id}', 'u1', 'amt'),
  preset('impact', 'Impact', 'clickid={click_id}&amount={payout}&ActionTrackerId={tx_id}', '{clickid}', '{amount}'),
  preset('partnerstack', 'PartnerStack', 'key={click_id}&amount={payout}&xid={tx_id}', '{click_id}', '{amount}'),
  preset('tune', 'Tune / HasOffers', 'transaction_id={click_id}&amount={payout}&adv_sub={tx_id}', '{transaction_id}', '{payout}'),
  preset('cake', 'Cake', 'reqid={click_id}&price={payout}&c={tx_id}', '#reqid#', '#price#'),
  preset('everflow', 'Everflow', 'transaction_id={click_id}&amount={payout}&advertiser_reference={tx_id}', '{transaction_id}', '{payout}'),
  preset('affise', 'Affise', 'clickid={click_id}&sum={payout}&action_id={tx_id}', '{clickid}', '{sum}'),
  preset('cellxpert', 'Cellxpert', 'cid={click_id}&payout={payout}&txid={tx_id}', '{cid}', '{payout}'),
  preset('flexoffers', 'FlexOffers', 'sid={click_id}&amount={payout}&order={tx_id}', 'sid', 'amount'),
  preset('pepperjam', 'Pepperjam / Ascend', 'SID={click_id}&AMOUNT={payout}&ORDER_ID={tx_id}', 'SID', 'AMOUNT'),
  preset('webgains', 'Webgains', 'click_id={click_id}&value={payout}&order_id={tx_id}', 'click_id', 'value'),
  preset('tradedoubler', 'Tradedoubler', 'epi={click_id}&orderValue={payout}&orderNumber={tx_id}', 'epi', 'orderValue'),
  preset('digistore24', 'Digistore24', 'cid={click_id}&amount={payout}&order_id={tx_id}', '{cid}', '{amount}'),
  preset('clickbank', 'ClickBank', 'tid={click_id}&amount={payout}&receipt={tx_id}', 'tid', 'amount'),
  preset('jvzoo', 'JVZoo', 'cjevent={click_id}&amount={payout}&transaction={tx_id}', 'cjevent', 'amount'),
  preset('warriorplus', 'WarriorPlus', 'cid={click_id}&amount={payout}&txid={tx_id}', 'cid', 'amount'),
  preset('buygoods', 'BuyGoods', 'subid={click_id}&payout={payout}&orderid={tx_id}', 'subid', 'payout'),
  preset('giddyup', 'GiddyUp', 'click_id={click_id}&payout={payout}&tx_id={tx_id}', 'click_id', 'payout'),
  preset('cpa-house', 'CPA.house', 'clickid={click_id}&payout={payout}&action_id={tx_id}', '{clickid}', '{payout}'),
  preset('terraleads', 'Terraleads', 'clickid={click_id}&status=1&payout={payout}&txid={tx_id}', '{clickid}', '{payout}'),
  preset('drcash', 'Dr.Cash', 'clickid={click_id}&status=1&payout={payout}&action_id={tx_id}', '{clickid}', '{payout}'),
  preset('luckyonline', 'LuckyOnline', 'click_id={click_id}&sum={payout}&goal_id={tx_id}', '{click_id}', '{sum}'),
  preset('adcombo', 'AdCombo', 'clickid={click_id}&payout={payout}&trans_id={tx_id}', '{clickid}', '{payout}'),
  preset('lemonad', 'LemonAD', 'click_id={click_id}&payout={payout}&order_id={tx_id}', '{click_id}', '{payout}'),
  preset('shakes', 'Shakes.pro', 'clickid={click_id}&payout={payout}&txid={tx_id}', '{clickid}', '{payout}'),
  preset('mobidea', 'Mobidea', 'clickid={click_id}&payout={payout}&goal={tx_id}', '{clickid}', '{payout}'),
  preset('crakrevenue', 'CrakRevenue', 'clickid={click_id}&payout={payout}&txid={tx_id}', '#clickid#', '#payout#'),
  preset('trafficcompany', 'TrafficCompany', 'click_id={click_id}&payout={payout}&tx={tx_id}', '{click_id}', '{payout}'),
  preset('adcash', 'AdCash CPA', 'cid={click_id}&payout={payout}&txid={tx_id}', '{cid}', '{payout}'),
  {
    id: 'custom',
    name: 'Custom / Generic',
    url_template: 'https://partner.example/postback?cid={click_id}&payout={payout}&txid={tx_id}&event={event_type}',
    network_click_token: '{click_id}',
    network_payout_token: '{payout}',
    notes: 'Generic BidShard macros. Replace host and query keys to match the partner.',
  },
];

/**
 * Lookup affiliate postback preset by id.
 */
export function affiliatePostbackById(id: string): AffiliatePostbackPreset | null {
  for (let i = 0; i < AFFILIATE_POSTBACK_PRESETS.length; i += 1) {
    if (AFFILIATE_POSTBACK_PRESETS[i].id === id) return AFFILIATE_POSTBACK_PRESETS[i];
  }
  return null;
}
