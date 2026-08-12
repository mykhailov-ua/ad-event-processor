export type ViewHandle = {
  destroy?: () => void;
};

export type RouteContext = {
  params: Record<string, string>;
  query: URLSearchParams;
  navigate: (path: string) => void;
};

export type ViewModule = {
  mount: (el: HTMLElement, ctx: RouteContext) => ViewHandle | void | null;
};

export type RouteDef = {
  path: string;
  shell?: boolean;
  load: () => Promise<ViewModule>;
};
