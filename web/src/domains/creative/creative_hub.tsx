import { HubLinkCard, HubLinkGrid } from '@/components/system/hub_link_card';
import { PageChrome } from '@/components/system/page_chrome';
import { CreativeNav } from '@/domains/creative/creative_nav';

const CREATIVE_LINKS = [
  {
    path: '/flows',
    title: 'Flows',
    description: 'Weighted lander and offer path definitions for campaign routing.',
  },
  {
    path: '/landers',
    title: 'Landers',
    description: 'External URLs and hosted lander assets with draft editor and publish.',
  },
  {
    path: '/offers',
    title: 'Offers',
    description: 'Offer URLs referenced from flow paths.',
  },
  {
    path: '/brands',
    title: 'Brands',
    description: 'Customer-scoped brands and rotating brand creatives.',
  },
  {
    path: '/supply',
    title: 'Supply',
    description: 'Sellers.json and ads.txt rows with server preview and export path.',
  },
  {
    path: '/domains',
    title: 'Domains',
    description: 'Custom tracking domain health, SSL, and probe status.',
  },
];

export function CreativeHub() {
  return (
    <PageChrome title="Creative and traffic">
      <CreativeNav />
      <HubLinkGrid>
        {CREATIVE_LINKS.map((item) => (
          <HubLinkCard key={item.path} {...item} />
        ))}
      </HubLinkGrid>
    </PageChrome>
  );
}
