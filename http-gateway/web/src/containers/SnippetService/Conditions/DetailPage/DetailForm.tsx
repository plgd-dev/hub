import React, { FC, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'
import get from 'lodash/get'
import { Controller } from 'react-hook-form'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import FormSelect, { selectAligns } from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import Tag from '@shared-ui/components/Atomic/Tag'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import ConditionFilter from '@shared-ui/components/Organisms/ConditionFilter/ConditionFilter'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import StatusTag from '@shared-ui/components/Atomic/StatusTag'

import { messages as g } from '@/containers/Global.i18n'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './DetailForm.types'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'

import { pages } from '@/routes'
import { formatText } from '@/containers/PendingCommands/DateFormat'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'

const DetailForm: FC<Props> = (props) => {
    const { formData, refs, resetIndex } = props
    const { formatMessage: _, formatDate, formatTime } = useIntl()
    const schema = useValidationsSchema('tab1')

    const {
        formState: { errors },
        register,
        updateField,
        watch,
        setValue,
        control,
        reset,
    } = useForm<Inputs>({
        defaultFormData: formData,
        errorKey: 'tab1',
        schema,
    })

    const navigate = useNavigate()

    const enabledOptions = useMemo(
        () => [
            { label: _(g.enabled), value: true },
            { label: _(g.disabled), value: false },
        ],
        [_]
    )

    console.log(watch())

    const resourceHrefFilter = watch('resourceHrefFilter')
    const resourceTypeFilter = watch('resourceTypeFilter')
    const jqExpressionFilterData = watch('jqExpressionFilter')
    const jqExpressionFilter = Array.isArray(jqExpressionFilterData) ? jqExpressionFilterData : [jqExpressionFilterData]

    const options = [
        { value: 'chocolate', label: 'Chocolate' },
        { value: 'strawberry', label: 'Strawberry' },
        { value: 'vanilla', label: 'Vanilla' },
    ]

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    return (
        <>
            <div ref={refs.general}>
                <Spacer type='mb-4'>
                    <Headline type='h5'>{_(g.general)}</Headline>
                </Spacer>

                <SimpleStripTable
                    leftColSize={6}
                    rightColSize={6}
                    rows={[
                        {
                            attribute: _(g.name),
                            value: (
                                <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                                    <FormInput
                                        {...register('name', { required: true, validate: (val) => val !== '' })}
                                        onBlur={(e) => updateField('name', e.target.value)}
                                        placeholder={_(g.name)}
                                    />
                                </FormGroup>
                            ),
                        },
                        {
                            attribute: _(g.status),
                            value: (
                                <FormGroup error={get(errors, 'enabled.message')} id='enabled'>
                                    <div>
                                        <Controller
                                            control={control}
                                            name='enabled'
                                            render={({ field: { onChange, value } }) => (
                                                <FormSelect
                                                    inlineStyle
                                                    align={selectAligns.RIGHT}
                                                    error={!!errors.name}
                                                    menuPortalTarget={document.body}
                                                    onChange={(options: OptionType) => {
                                                        const v = options.value
                                                        onChange(v)
                                                        updateField('enabled', v)
                                                    }}
                                                    options={enabledOptions}
                                                    size='small'
                                                    value={value !== undefined ? enabledOptions.filter((v: { value: boolean }) => value === v.value) : []}
                                                />
                                            )}
                                        />
                                    </div>
                                </FormGroup>
                            ),
                        },
                        {
                            attribute: _(confT.configSelect),
                            value: (
                                <Tag
                                    onClick={() =>
                                        navigate(
                                            generatePath(pages.CONDITIONS.RESOURCES_CONFIG.DETAIL.LINK, {
                                                resourcesConfigId: formData.configurationId,
                                                tab: '',
                                            })
                                        )
                                    }
                                    variant={tagVariants.BLUE}
                                >
                                    <IconLink /> &nbsp;{_(confT.configLink)}
                                </Tag>
                            ),
                        },
                        {
                            attribute: _(g.lastModified),
                            value: <FormInput disabled value={formatText(formData.timestamp, formatDate, formatTime)} />,
                        },
                    ]}
                />
            </div>
            <Spacer ref={refs.general} type='pt-8'>
                <Spacer type='mb-6'>
                    <Headline type='h5'>{_(g.filters)}</Headline>
                    <p style={{ margin: '4px 0 0 0' }}>Short description...</p>

                    <Spacer ref={refs.filterDeviceId} type='pt-6'>
                        <ConditionFilter
                            status={
                                <StatusTag lowercase={false} variant='success'>
                                    {_(g.setUp)}
                                </StatusTag>
                            }
                            title={_(confT.deviceIdFilter)}
                        >
                            <FormGroup id='devicesId'>
                                <FormLabel text={_(confT.selectDevices)} />
                                <FormSelect
                                    creatable
                                    isMulti
                                    menuPortalTarget={document.body}
                                    name='devicesId'
                                    options={options}
                                    placeholder={_(g.selectOrCreate)}
                                />
                            </FormGroup>
                        </ConditionFilter>
                    </Spacer>

                    <Spacer ref={refs.filterResourceType} type='pt-2'>
                        <ConditionFilter
                            listName={_(confT.listOfSelectedResourceType)}
                            listOfItems={resourceTypeFilter}
                            onItemDelete={(key) => {
                                const newVal = resourceTypeFilter.filter((_, i) => i !== key)
                                setValue('resourceTypeFilter', newVal)
                                updateField('resourceTypeFilter', newVal)
                            }}
                            status={
                                <StatusTag lowercase={false} variant={resourceTypeFilter.length > 0 ? 'success' : 'normal'}>
                                    {resourceTypeFilter.length > 0 ? _(g.setUp) : _(g.notSet)}
                                </StatusTag>
                            }
                            title={_(confT.resourceTypeFilter)}
                        >
                            <FormGroup id='devicesId'>
                                <FormLabel text={_(confT.addManualData)} />
                                <FormInput
                                    compactFormComponentsView={false}
                                    onKeyPress={(e) => {
                                        if (e.key === 'Enter') {
                                            const newVal = [...resourceTypeFilter, e.target.value]
                                            setValue('resourceTypeFilter', newVal)
                                            updateField('resourceTypeFilter', newVal)
                                        }
                                    }}
                                />
                            </FormGroup>
                        </ConditionFilter>
                    </Spacer>

                    <Spacer ref={refs.filterResourceHref} type='pt-2'>
                        <ConditionFilter
                            listName={_(confT.listOfSelectedHrefFilter)}
                            listOfItems={resourceHrefFilter}
                            onItemDelete={(key) => {
                                const newVal = resourceHrefFilter.filter((_, i) => i !== key)
                                setValue('resourceHrefFilter', newVal)
                                updateField('resourceHrefFilter', newVal)
                            }}
                            status={
                                <StatusTag lowercase={false} variant={resourceHrefFilter.length > 0 ? 'success' : 'normal'}>
                                    {resourceHrefFilter.length > 0 ? _(g.setUp) : _(g.notSet)}
                                </StatusTag>
                            }
                            title={_(confT.resourceHrefFilter)}
                        >
                            <FormGroup id='devicesId'>
                                <FormLabel text={_(confT.addManualData)} />
                                <FormInput
                                    compactFormComponentsView={false}
                                    onKeyPress={(e) => {
                                        if (e.key === 'Enter') {
                                            const newVal = [...resourceHrefFilter, e.target.value]
                                            setValue('resourceHrefFilter', newVal)
                                            updateField('resourceHrefFilter', newVal)
                                        }
                                    }}
                                />
                            </FormGroup>
                        </ConditionFilter>
                    </Spacer>

                    <Spacer ref={refs.filterJqExpression} type='pt-2'>
                        <ConditionFilter
                            listName={_(confT.listOfSelectedJqExpression)}
                            listOfItems={jqExpressionFilter}
                            onItemDelete={(key) => {
                                const newVal = jqExpressionFilter.filter((_, i) => i !== key)
                                setValue('jqExpressionFilter', newVal)
                                updateField('jqExpressionFilter', newVal)
                            }}
                            status={
                                <StatusTag lowercase={false} variant={jqExpressionFilter.length > 0 ? 'success' : 'normal'}>
                                    {jqExpressionFilter.length > 0 ? _(g.setUp) : _(g.notSet)}
                                </StatusTag>
                            }
                            title={_(confT.jqExpression)}
                        >
                            <FormGroup id='devicesId'>
                                <FormLabel text={_(confT.addManualData)} />
                                <FormInput
                                    compactFormComponentsView={false}
                                    onKeyPress={(e) => {
                                        if (e.key === 'Enter') {
                                            const newVal = [...jqExpressionFilter, e.target.value]
                                            setValue('jqExpressionFilter', newVal)
                                            updateField('jqExpressionFilter', newVal)
                                        }
                                    }}
                                />
                            </FormGroup>
                        </ConditionFilter>
                    </Spacer>

                    <Spacer ref={refs.accessToken} type='pt-8'>
                        <Headline type='h5'>{_(confT.APIAccessToken)}</Headline>
                        <p style={{ margin: '4px 0 0 0' }}>Short description...</p>

                        <Spacer type='pt-6'>
                            <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                                <FormTextarea
                                    {...register('apiAccessToken', { required: true, validate: (val) => val !== '' })}
                                    onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                                    style={{ height: 450 }}
                                />
                            </FormGroup>
                        </Spacer>
                    </Spacer>
                </Spacer>
            </Spacer>
        </>
    )
}

DetailForm.displayName = 'DetailForm'

export default DetailForm
