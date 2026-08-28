import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  FIRST_CAMPAIGN_WIZARD_STEPS,
  buildWizardClickURL,
  bundledTrafficTemplateForSource,
  canLeaveFirstCampaignWizardStep,
  firstCampaignWizardStepLabel,
  nextFirstCampaignWizardStep,
  prevFirstCampaignWizardStep,
  validateFirstCampaignBasics,
} from './first_campaign_wizard_model.js';

describe('first_campaign_wizard_model', () => {
  it('orders steps from campaign through done', () => {
    assert.deepEqual(FIRST_CAMPAIGN_WIZARD_STEPS, [
      'campaign',
      'traffic',
      'lander',
      'test_click',
      'test_postback',
      'done',
    ]);
    assert.equal(firstCampaignWizardStepLabel('traffic'), 'Traffic source');
    assert.equal(nextFirstCampaignWizardStep('lander'), 'test_click');
    assert.equal(prevFirstCampaignWizardStep('test_postback'), 'test_click');
    assert.equal(nextFirstCampaignWizardStep('done'), null);
  });

  it('maps common traffic sources to bundled integration slugs', () => {
    assert.equal(bundledTrafficTemplateForSource('meta-facebook'), 'traffic_facebook');
    assert.equal(bundledTrafficTemplateForSource('tiktok-ads'), 'traffic_tiktok');
    assert.equal(bundledTrafficTemplateForSource('direct-custom'), null);
  });

  it('builds click URL with campaign id and macros', () => {
    const url = buildWizardClickURL('https://trk.example/click', 'camp-uuid', 'meta-facebook');
    assert.match(url, /campaign_id=camp-uuid/);
    assert.match(url, /fbclid=/);
  });

  it('validates basics form', () => {
    const customerId = '00000000-0000-4000-8000-000000000001';
    assert.equal(
      validateFirstCampaignBasics({
        customerId,
        templateId: 'tpl-1',
        name: 'Test',
        budgetInput: '',
      }),
      null
    );
    assert.match(
      validateFirstCampaignBasics({
        customerId: '',
        templateId: 'tpl-1',
        name: 'Test',
        budgetInput: '',
      }) ?? '',
      /Customer UUID/
    );
    assert.match(
      validateFirstCampaignBasics({
        customerId,
        templateId: '',
        name: 'Test',
        budgetInput: '',
      }) ?? '',
      /template/
    );
  });

  it('blocks post-campaign steps until campaign exists', () => {
    assert.equal(canLeaveFirstCampaignWizardStep('campaign', ''), true);
    assert.equal(canLeaveFirstCampaignWizardStep('traffic', ''), false);
    assert.equal(canLeaveFirstCampaignWizardStep('traffic', 'camp-1'), true);
  });
});
