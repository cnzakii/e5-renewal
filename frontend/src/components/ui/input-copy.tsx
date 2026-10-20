import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { Hint } from '@/components/common'
import { Button } from './button'
import { Input } from './input'

export function InputCopy({ value, label }: { value: string; label: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
    } catch {
      toast.error(t('ui.could_not_copy_select_the_text_and_copy_it_manually'))
    }
  }
  return (
    <div className="relative">
      <Input
        readOnly
        value={value}
        aria-label={label}
        className="pe-12 font-mono text-xs"
      />
      <Hint label={t(copied ? 'ui.copied' : 'ui.copy')}>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={t('ui.copy') + ' ' + label}
          className="absolute end-0 top-0"
          onBlur={() => setCopied(false)}
          onClick={() => void copy()}
        >
          {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
        </Button>
      </Hint>
    </div>
  )
}
