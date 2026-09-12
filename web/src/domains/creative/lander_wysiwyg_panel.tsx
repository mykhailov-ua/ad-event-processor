import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { FilterField } from '@/shell/filter_panel';
import { creativePanelError } from '@/domains/creative/creative_nav';
import type { LanderBlock } from '@/domains/creative/lander_block_model';
import { LANDER_BLOCK_OPTIONS } from '@/domains/creative/lander_block_model';
import { adminKit, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type LanderWysiwygPanelProps = {
  blocks: LanderBlock[];
  previewHtml: string;
  dirty?: boolean;
  saving?: boolean;
  error?: Error;
  disabled?: boolean;
  onBlocksChange: (blocks: LanderBlock[]) => void;
  onAddBlock: (type: LanderBlock['type']) => void;
  onSave: () => void;
};

function updateBlockAt(
  blocks: LanderBlock[],
  index: number,
  patch: Partial<LanderBlock>
): LanderBlock[] {
  return blocks.map((block, blockIndex) =>
    blockIndex === index ? ({ ...block, ...patch } as LanderBlock) : block
  );
}

function moveBlock(blocks: LanderBlock[], index: number, direction: -1 | 1): LanderBlock[] {
  const target = index + direction;
  if (target < 0 || target >= blocks.length) {
    return blocks;
  }
  const next = [...blocks];
  const [item] = next.splice(index, 1);
  next.splice(target, 0, item);
  return next;
}

export function LanderWysiwygPanel({
  blocks,
  previewHtml,
  dirty = false,
  saving = false,
  error,
  disabled = false,
  onBlocksChange,
  onAddBlock,
  onSave,
}: LanderWysiwygPanelProps) {
  return (
    <section className="grid gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className={adminTypography.sectionTitle}>Block editor</h2>
        {dirty ? (
          <span className={cn(adminTypography.captionPlain, 'text-muted-foreground')}>
            Unsaved block changes
          </span>
        ) : null}
      </div>

      <div className="flex flex-wrap items-end gap-2">
        <FilterField htmlFor="lander-add-block" label="Add block">
          <Select
            disabled={disabled}
            onValueChange={(value) => onAddBlock(value as LanderBlock['type'])}
          >
            <SelectTrigger className="w-48" id="lander-add-block">
              <SelectValue placeholder="Choose block type" />
            </SelectTrigger>
            <SelectContent>
              {LANDER_BLOCK_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>
        <PrimaryActionButton
          disabled={disabled || !dirty}
          loading={saving}
          onClick={onSave}
          type="button"
        >
          Save blocks to index.html
        </PrimaryActionButton>
      </div>

      {error ? creativePanelError(error, 'Could not save block layout') : null}

      <div className="grid gap-3">
        {blocks.map((block, index) => (
          <article
            key={block.id}
            className={cn('grid gap-3 border border-border/50 p-4', adminKit.panelRadius)}
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className={adminTypography.bodyMuted}>
                Block {index + 1}: {block.type}
              </p>
              <div className="flex flex-wrap gap-2">
                <Button
                  disabled={disabled || index === 0}
                  onClick={() => onBlocksChange(moveBlock(blocks, index, -1))}
                  type="button"
                  variant="outline"
                >
                  Move up
                </Button>
                <Button
                  disabled={disabled || index === blocks.length - 1}
                  onClick={() => onBlocksChange(moveBlock(blocks, index, 1))}
                  type="button"
                  variant="outline"
                >
                  Move down
                </Button>
                <Button
                  disabled={disabled || blocks.length <= 1}
                  onClick={() =>
                    onBlocksChange(blocks.filter((_, blockIndex) => blockIndex !== index))
                  }
                  type="button"
                  variant="outline"
                >
                  Remove
                </Button>
              </div>
            </div>

            {block.type === 'hero' ? (
              <div className="grid gap-3">
                <div className="grid gap-1">
                  <Label htmlFor={`hero-title-${block.id}`}>Title</Label>
                  <Input
                    disabled={disabled}
                    id={`hero-title-${block.id}`}
                    value={block.title}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { title: event.target.value }))
                    }
                  />
                </div>
                <div className="grid gap-1">
                  <Label htmlFor={`hero-subtitle-${block.id}`}>Subtitle</Label>
                  <Input
                    disabled={disabled}
                    id={`hero-subtitle-${block.id}`}
                    value={block.subtitle}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { subtitle: event.target.value }))
                    }
                  />
                </div>
              </div>
            ) : null}

            {block.type === 'cta' ? (
              <div className="grid gap-3">
                <div className="grid gap-1">
                  <Label htmlFor={`cta-label-${block.id}`}>Label</Label>
                  <Input
                    disabled={disabled}
                    id={`cta-label-${block.id}`}
                    value={block.label}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { label: event.target.value }))
                    }
                  />
                </div>
                <div className="grid gap-1">
                  <Label htmlFor={`cta-href-${block.id}`}>Link URL</Label>
                  <Input
                    disabled={disabled}
                    id={`cta-href-${block.id}`}
                    value={block.href}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { href: event.target.value }))
                    }
                  />
                </div>
              </div>
            ) : null}

            {block.type === 'image' ? (
              <div className="grid gap-3">
                <div className="grid gap-1">
                  <Label htmlFor={`image-src-${block.id}`}>Image URL</Label>
                  <Input
                    disabled={disabled}
                    id={`image-src-${block.id}`}
                    value={block.src}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { src: event.target.value }))
                    }
                  />
                </div>
                <div className="grid gap-1">
                  <Label htmlFor={`image-alt-${block.id}`}>Alt text</Label>
                  <Input
                    disabled={disabled}
                    id={`image-alt-${block.id}`}
                    value={block.alt}
                    onChange={(event) =>
                      onBlocksChange(updateBlockAt(blocks, index, { alt: event.target.value }))
                    }
                  />
                </div>
              </div>
            ) : null}

            {block.type === 'html' ? (
              <div className="grid gap-1">
                <Label htmlFor={`html-content-${block.id}`}>Raw HTML</Label>
                <Textarea
                  className="min-h-32 font-mono"
                  disabled={disabled}
                  id={`html-content-${block.id}`}
                  value={block.content}
                  onChange={(event) =>
                    onBlocksChange(updateBlockAt(blocks, index, { content: event.target.value }))
                  }
                />
              </div>
            ) : null}
          </article>
        ))}
      </div>

      <div className="grid gap-2">
        <h3 className={adminTypography.sectionTitle}>Block preview</h3>
        <iframe
          className={cn(
            'min-h-[360px] w-full border border-border/50 bg-muted/40',
            adminKit.panelRadius
          )}
          srcDoc={previewHtml}
          title="Block layout preview"
        />
      </div>
    </section>
  );
}
