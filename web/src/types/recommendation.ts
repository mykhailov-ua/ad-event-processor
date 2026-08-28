export type RecommendationAction = {
  id?: string;
  label?: string;
};

export type RecommendationCard = {
  id: string;
  type: string;
  title: string;
  detail: string;
  campaign_id?: string;
  confidence?: number;
  actions?: RecommendationAction[];
};
