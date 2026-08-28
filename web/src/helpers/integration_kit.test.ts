import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildInboundS2SBodyTemplate,
  buildInboundS2SCurl,
  buildTrackPostbackURL,
  normalizeTrackingHost,
  trafficGuideSummary,
} from './integration_kit.js';

describe('integration_kit', () => {
  it('normalizes tracking host', () => {
    assert.equal(normalizeTrackingHost('https://trk.example.com/'), 'trk.example.com');
    assert.equal(normalizeTrackingHost('trk.example.com'), 'trk.example.com');
  });

  it('builds postback URL', () => {
    assert.equal(buildTrackPostbackURL('trk.example.com'), 'https://trk.example.com/track');
    assert.equal(buildTrackPostbackURL(''), 'https://track.example/track');
  });

  it('builds inbound S2S body with campaign id', () => {
    const body = buildInboundS2SBodyTemplate('camp-uuid');
    assert.match(body, /"campaign_id": "camp-uuid"/);
    assert.match(body, /"click_id": "\{click_id\}"/);
    assert.match(body, /"type": "conversion"/);
  });

  it('builds curl with Content-Length', () => {
    const curl = buildInboundS2SCurl('https://trk.example/track', 'camp-1');
    assert.match(curl, /Content-Length:/);
    assert.match(curl, /camp-1/);
    assert.match(curl, /REPLACE_CLICK_ID/);
  });

  it('guide summary mentions click and postback split', () => {
    const s = trafficGuideSummary();
    assert.match(s, /GET \/click/);
    assert.match(s, /POST \/track/);
  });
});
