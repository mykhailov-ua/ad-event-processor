#!/usr/bin/env node
import { runCampaignListHotPathBench } from '@/lib/perf/benches/campaign_list.ts';
import { runDashboardHotPathBench } from '@/lib/perf/benches/dashboard.ts';
import { runNavFilterHotPathBench } from '@/lib/perf/benches/nav_filter.ts';

console.log('admin web bench: hot path (campaign list + dashboard + nav filter)');

runCampaignListHotPathBench();
runDashboardHotPathBench();
runNavFilterHotPathBench();

console.log('admin web bench: hot path PASSED');
