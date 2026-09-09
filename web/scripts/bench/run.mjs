#!/usr/bin/env node
import { runCampaignListExportColdPathBench } from '@/lib/perf/benches/campaign_list_export.ts';

console.log('admin web bench: cold path (campaign list CSV export)');
runCampaignListExportColdPathBench();
console.log('admin web bench: cold path PASSED');
