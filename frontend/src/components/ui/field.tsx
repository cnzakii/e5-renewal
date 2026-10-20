import type { ComponentProps } from 'react'
import { cn } from '@/lib/utils'
import { Label } from '@/components/ui/label'

function FieldGroup({ className, ...props }: ComponentProps<'div'>) {
  return <div className={cn('grid gap-4', className)} {...props} />
}

function FieldSet({ className, ...props }: ComponentProps<'fieldset'>) {
  return <fieldset className={cn('grid min-w-0 gap-4', className)} {...props} />
}

function FieldLegend({
  className,
  children,
  ...props
}: ComponentProps<'legend'>) {
  return (
    <legend
      className={cn('mb-4 w-full text-sm font-semibold', className)}
      {...props}
    >
      <span className="flex items-center gap-2">{children}</span>
    </legend>
  )
}

function Field({ className, ...props }: ComponentProps<'div'>) {
  return <div className={cn('grid gap-2', className)} {...props} />
}

function FieldLabel({ className, ...props }: ComponentProps<typeof Label>) {
  return <Label className={cn('leading-none', className)} {...props} />
}

function FieldDescription({ className, ...props }: ComponentProps<'p'>) {
  return (
    <p className={cn('text-sm text-muted-foreground', className)} {...props} />
  )
}

export {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSet,
  FieldLegend,
}
