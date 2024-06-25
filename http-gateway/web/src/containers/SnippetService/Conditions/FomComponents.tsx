import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Loadable from '@shared-ui/components/Atomic/Loadable'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import ConditionFilter from '@shared-ui/components/Organisms/ConditionFilter'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import Button from '@shared-ui/components/Atomic/Button'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { useDevicesList } from '@/containers/Devices/hooks'

type Props = {
    isActivePage?: boolean
    setValue: any
    updateField: any
    watch: any
}

const defaultProps: Partial<Props> = {
    isActivePage: true,
}

export const Step2FormComponent: FC<Props> = (props) => {
    const { isActivePage, watch, setValue, updateField } = { ...defaultProps, ...props }
    const { formatMessage: _ } = useIntl()

    const [options, setOptions] = useState<OptionType[]>([])
    const [defaultOptions, setDefaultOptions] = useState<OptionType[]>([])

    const deviceIdFilterVal: string[] = watch('deviceIdFilter')
    const resourceHrefFilterVal: string[] = watch('resourceHrefFilter')
    const resourceTypeFilterVal: string[] = watch('resourceTypeFilter')
    const jqExpressionFilterVal: string = watch('jqExpressionFilter')

    const [resourceTypeValue, setResourceTypeValue] = useState<string>('')
    const [resourceHrefValue, setResourceHrefValue] = useState<string>('')

    const deviceIdFilter: string[] = useMemo(() => deviceIdFilterVal || [], [deviceIdFilterVal])
    const resourceHrefFilter: string[] = useMemo(() => resourceHrefFilterVal || [], [resourceHrefFilterVal])
    const resourceTypeFilter: string[] = useMemo(() => resourceTypeFilterVal || [], [resourceTypeFilterVal])
    const jqExpressionFilter = useMemo(() => jqExpressionFilterVal || '', [jqExpressionFilterVal])

    const { data: devicesData, loading } = useDevicesList(isActivePage)

    useEffect(() => {
        const o: OptionType[] = devicesData?.map((device: { id: string; name: string }) => ({ value: device.id, label: `${device.name} - ${device.id}` }))
        setDefaultOptions(o)
        setOptions(o)
    }, [devicesData])

    const value = useMemo(() => deviceIdFilter.map((id: string) => options?.find((o) => o.value === id) || { value: id, label: id }), [deviceIdFilter, options])

    return (
        <>
            <Loadable condition={!loading}>
                <ConditionFilter
                    listName={_(confT.listOfSelectedDevices)}
                    listOfItems={deviceIdFilter.map((id) => options?.find((o) => o.value === id)?.label || id)}
                    onItemDelete={(key) => {
                        const newItems = deviceIdFilter.filter((_, i) => i !== key)
                        setValue('deviceIdFilter', newItems)
                        updateField('deviceIdFilter', newItems)
                    }}
                    status={
                        <StatusTag lowercase={false} variant={deviceIdFilter.length > 0 ? 'success' : 'normal'}>
                            {deviceIdFilter.length > 0 ? _(g.setUp) : _(g.notSet)}
                        </StatusTag>
                    }
                    title={_(confT.deviceIdFilter)}
                >
                    <FormGroup id='deviceIdFilter'>
                        <FormLabel text={_(confT.selectDevices)} />
                        <FormSelect
                            checkboxOptions
                            creatable
                            isMulti
                            footerLinksLeft={[
                                {
                                    title: _(g.reset),
                                    onClick: () => {
                                        setOptions(defaultOptions)
                                        setValue('deviceIdFilter', [])
                                    },
                                },
                                {
                                    title: _(g.done),
                                    variant: 'primary',
                                    onClick: (values: OptionType[]) => {
                                        const value = values.map((v) => v.value)
                                        setValue('deviceIdFilter', value)
                                        updateField('deviceIdFilter', value)
                                    },
                                },
                            ]}
                            i18n={{
                                itemSelected: _(g.deviceSelected),
                                itemsSelected: _(g.devicesSelected),
                            }}
                            menuPortalTarget={document.getElementById('modal-root')}
                            menuZIndex={100}
                            name='deviceIdFilter'
                            onChange={(values: OptionType[]) => {
                                const value = values.map((v) => v.value)
                                setValue('deviceIdFilter', value)
                                updateField('deviceIdFilter', values)
                            }}
                            onCreateOption={(value: string | number) => {
                                setOptions((prev) => [...prev, { value: value.toString(), label: value.toString() }])
                            }}
                            options={options}
                            placeholder={_(g.selectOrCreate)}
                            value={value}
                        />
                    </FormGroup>
                </ConditionFilter>
            </Loadable>

            <Spacer type='pt-2'>
                <ConditionFilter
                    listName={_(confT.listOfSelectedResourceType)}
                    listOfItems={resourceTypeFilter}
                    onItemDelete={(key) => {
                        const newVal = resourceTypeFilter?.filter((_, i) => i !== key)
                        setValue('resourceTypeFilter', newVal)
                        updateField('resourceTypeFilter', newVal)
                    }}
                    status={
                        <StatusTag lowercase={false} variant={resourceTypeFilter?.length > 0 ? 'success' : 'normal'}>
                            {resourceTypeFilter?.length > 0 ? _(g.setUp) : _(g.notSet)}
                        </StatusTag>
                    }
                    title={_(confT.resourceTypeFilter)}
                >
                    <div style={{ display: 'flex', alignItems: 'flex-end', gap: '8px' }}>
                        <FormGroup id='resourceTypeFilter' marginBottom={false} style={{ flex: '1 1 auto' }}>
                            <FormLabel text={_(confT.addManualData)} />
                            <FormInput
                                compactFormComponentsView={false}
                                onChange={(e) => setResourceTypeValue(e.target.value)}
                                onKeyPress={(e) => {
                                    if (e.key === 'Enter' && e.target.value !== '') {
                                        e.preventDefault()
                                        const newVal = [...resourceTypeFilter, e.target.value]
                                        setValue('resourceTypeFilter', newVal)
                                        updateField('resourceTypeFilter', newVal)
                                        setResourceTypeValue('')
                                    }
                                }}
                                value={resourceTypeValue}
                            />
                        </FormGroup>
                        <Button
                            disabled={resourceTypeValue === ''}
                            icon={<IconPlus />}
                            onClick={() => {
                                const newVal = [...resourceTypeFilter, resourceTypeValue]
                                setValue('resourceTypeFilter', newVal)
                                updateField('resourceTypeFilter', newVal)
                                setResourceTypeValue('')
                            }}
                            size='small'
                            style={{
                                position: 'relative',
                                bottom: '2px',
                            }}
                            variant='secondary'
                        >
                            {_(g.add)}
                        </Button>
                    </div>
                </ConditionFilter>
            </Spacer>

            <Spacer type='pt-2'>
                <ConditionFilter
                    listName={_(confT.listOfSelectedHrefFilter)}
                    listOfItems={resourceHrefFilter}
                    onItemDelete={(key) => {
                        const newVal = resourceHrefFilter?.filter((_, i) => i !== key)
                        setValue('resourceHrefFilter', newVal)
                        updateField('resourceHrefFilter', newVal)
                    }}
                    status={
                        <StatusTag lowercase={false} variant={resourceHrefFilter?.length > 0 ? 'success' : 'normal'}>
                            {resourceHrefFilter?.length > 0 ? _(g.setUp) : _(g.notSet)}
                        </StatusTag>
                    }
                    title={_(confT.resourceHrefFilter)}
                >
                    <div style={{ display: 'flex', alignItems: 'flex-end', gap: '8px' }}>
                        <FormGroup id='resourceHrefFilter' marginBottom={false} style={{ flex: '1 1 auto' }}>
                            <FormLabel text={_(confT.addManualData)} />
                            <FormInput
                                compactFormComponentsView={false}
                                onChange={(e) => setResourceHrefValue(e.target.value)}
                                onKeyPress={(e) => {
                                    if (e.key === 'Enter' && e.target.value !== '') {
                                        e.preventDefault()
                                        const newVal = [...resourceHrefFilter, e.target.value]
                                        setValue('resourceHrefFilter', newVal)
                                        updateField('resourceHrefFilter', newVal)
                                        setResourceHrefValue('')
                                    }
                                }}
                                value={resourceHrefValue}
                            />
                        </FormGroup>
                        <Button
                            disabled={resourceHrefValue === ''}
                            icon={<IconPlus />}
                            onClick={() => {
                                const newVal = [...resourceHrefFilter, resourceHrefValue]
                                setValue('resourceHrefFilter', newVal)
                                updateField('resourceHrefFilter', newVal)
                                setResourceHrefValue('')
                            }}
                            size='small'
                            style={{
                                position: 'relative',
                                bottom: '2px',
                            }}
                            variant='secondary'
                        >
                            {_(g.add)}
                        </Button>
                    </div>
                </ConditionFilter>
            </Spacer>

            <Spacer type='pt-2'>
                <ConditionFilter
                    listName={_(confT.listOfSelectedJqExpression)}
                    status={
                        <StatusTag lowercase={false} variant={jqExpressionFilter !== '' ? 'success' : 'normal'}>
                            {jqExpressionFilter !== '' ? _(g.setUp) : _(g.notSet)}
                        </StatusTag>
                    }
                    title={_(confT.jqExpression)}
                >
                    <FormGroup id='jqExpressionFilter'>
                        <FormLabel text={_(confT.addManualData)} />
                        <FormInput
                            compactFormComponentsView={false}
                            onChange={(e) => {
                                setValue('jqExpressionFilter', e.target.value)
                                updateField('jqExpressionFilter', e.target.value)
                            }}
                            onKeyPress={(e) => {
                                if (e.key === 'Enter') {
                                    e.preventDefault()
                                    setValue('jqExpressionFilter', e.target.value)
                                    updateField('jqExpressionFilter', e.target.value)
                                }
                            }}
                            value={jqExpressionFilter}
                        />
                    </FormGroup>
                </ConditionFilter>
            </Spacer>
        </>
    )
}
