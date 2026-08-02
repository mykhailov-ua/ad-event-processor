import { createElement as lucideCreate } from 'lucide';
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowDownRight,
  ArrowUpDown,
  ArrowUpRight,
  BarChart3,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Clock,
  Copy,
  CreditCard,
  Download,
  Edit3,
  ExternalLink,
  Eye,
  FileSpreadsheet,
  FileText,
  Filter,
  Globe,
  HelpCircle,
  Info,
  Key,
  Layers,
  LayoutDashboard,
  LogOut,
  Megaphone,
  Moon,
  MoreVertical,
  PanelLeftClose,
  PanelLeftOpen,
  Palette,
  Pause,
  Play,
  Plus,
  RefreshCw,
  Search,
  Server,
  Settings,
  Shield,
  SlidersHorizontal,
  Sun,
  ToggleLeft,
  Trash2,
  TrendingUp,
  Users,
  X,
  Zap,
} from 'lucide';

/** @type {Record<string, import('lucide').IconNode>} */
const ICONS = {
  'layout-dashboard': LayoutDashboard,
  megaphone: Megaphone,
  users: Users,
  'credit-card': CreditCard,
  'bar-chart': BarChart3,
  'file-text': FileText,
  activity: Activity,
  settings: Settings,
  globe: Globe,
  server: Server,
  'toggle-left': ToggleLeft,
  'log-out': LogOut,
  sun: Sun,
  moon: Moon,
  search: Search,
  'panel-left-close': PanelLeftClose,
  'panel-left-open': PanelLeftOpen,
  palette: Palette,
  plus: Plus,
  trash: Trash2,
  refresh: RefreshCw,
  download: Download,
  filter: Filter,
  check: Check,
  x: X,
  'alert-triangle': AlertTriangle,
  'alert-circle': AlertCircle,
  info: Info,
  'chevron-right': ChevronRight,
  'chevron-down': ChevronDown,
  'chevron-up': ChevronUp,
  'arrow-up-down': ArrowUpDown,
  sliders: SlidersHorizontal,
  layers: Layers,
  shield: Shield,
  zap: Zap,
  'trending-up': TrendingUp,
  copy: Copy,
  'external-link': ExternalLink,
  'file-spreadsheet': FileSpreadsheet,
  play: Play,
  pause: Pause,
  'help-circle': HelpCircle,
  edit: Edit3,
  clock: Clock,
  'arrow-up-right': ArrowUpRight,
  'arrow-down-right': ArrowDownRight,
  eye: Eye,
  'more-vertical': MoreVertical,
  'check-circle': CheckCircle2,
  key: Key,
};

/**
 * @param {string} name
 * @param {{ size?: number, className?: string, strokeWidth?: number }} [opts]
 * @returns {SVGElement|HTMLSpanElement}
 */
export function renderIcon(name, opts = {}) {
  const def = ICONS[name];
  if (!def) {
    const span = document.createElement('span');
    span.className = 'icon icon--missing';
    span.setAttribute('aria-hidden', 'true');
    return span;
  }
  const svg = lucideCreate(def, {
    width: opts.size ?? 16,
    height: opts.size ?? 16,
    'stroke-width': opts.strokeWidth ?? 1.75,
    class: ['icon', opts.className].filter(Boolean).join(' '),
  });
  svg.setAttribute('aria-hidden', 'true');
  return svg;
}
