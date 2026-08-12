/** RTB admin API DTOs — mirror adminapi/rtb_handlers.go */

export type RtbDealDTO = {
  id: number;
  deal_id: string;
  floor_micro: number;
  geo_mask: number;
  cat_mask: number;
  pacing: string;
  seats: number;
  customer_id: string;
  created_at: string;
  updated_at: string;
};

export type RtbDealCreateSpec = {
  deal_id: string;
  floor_micro: number;
  geo_mask?: number;
  cat_mask?: number;
  pacing?: string;
  seats?: number;
  customer_id: string;
};

export type RtbDealUpdateSpec = Partial<RtbDealCreateSpec>;

export type RtbFloorSuggestionDTO = {
  placement_id: string;
  deal_id: string;
  current_floor_micro: number;
  suggested_floor_micro: number;
  win_rate: number;
  sample_n: number;
  floor_bucket_micro: number;
  computed_at: string;
};

export type RtbFloorsApplyResult = {
  dry_run: boolean;
  applied: number;
  suggestions: RtbFloorSuggestionDTO[];
  outbox_rows: number;
};
