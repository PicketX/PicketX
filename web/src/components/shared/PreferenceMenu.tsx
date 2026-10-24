// SPDX-License-Identifier: Apache-2.0
import { useId } from 'react'
import type { ReactNode } from 'react'
import { Check, ChevronDown } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { Button } from '@/components/ui/button'

type Option = { value: string; label: string; description?: string; icon?: ReactNode; lang?: string }

type Props = {
  label: string
  icon: ReactNode
  value: string
  options: Option[]
  onValueChange: (value: string) => void
}

export function PreferenceMenu({ label, icon, value, options, onValueChange }: Props) {
  const labelId = useId()
  return <DropdownMenu.Root>
    <DropdownMenu.Trigger asChild>
      <Button type="button" variant="ghost" className="preference-trigger" aria-label={label} title={`${label}: ${options.find(option => option.value === value)?.label ?? value}`}>
        {icon}<span>{label}</span><ChevronDown aria-hidden="true" className="preference-chevron" />
      </Button>
    </DropdownMenu.Trigger>
    <DropdownMenu.Portal>
      <DropdownMenu.Content className="preference-menu" align="end" sideOffset={8} collisionPadding={12} aria-labelledby={labelId}>
        <DropdownMenu.Label className="preference-heading" id={labelId}>{label}</DropdownMenu.Label>
        <DropdownMenu.RadioGroup value={value} onValueChange={onValueChange} aria-labelledby={labelId}>
          {options.map(option => <DropdownMenu.RadioItem className="preference-option" key={option.value} value={option.value} textValue={option.label}>
            {option.icon}
            <span className="preference-option-copy"><span lang={option.lang}>{option.label}</span>{option.description && <small>{option.description}</small>}</span>
            <span className="preference-check"><DropdownMenu.ItemIndicator><Check aria-hidden="true" /></DropdownMenu.ItemIndicator></span>
          </DropdownMenu.RadioItem>)}
        </DropdownMenu.RadioGroup>
      </DropdownMenu.Content>
    </DropdownMenu.Portal>
  </DropdownMenu.Root>
}
