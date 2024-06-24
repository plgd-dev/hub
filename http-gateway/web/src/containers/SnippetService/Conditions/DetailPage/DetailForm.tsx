import React, { FC, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate } from 'react-router-dom'
import get from 'lodash/get'
import { Controller } from 'react-hook-form'
import pick from 'lodash/pick'
import isFunction from 'lodash/isFunction'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Headline from '@shared-ui/components/Atomic/Headline'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import Tag from '@shared-ui/components/Atomic/Tag'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'
import Switch from '@shared-ui/components/Atomic/Switch'

import { messages as g } from '@/containers/Global.i18n'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './DetailForm.types'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import { formatText } from '@/containers/PendingCommands/DateFormat'
import { Step2FormComponent } from '@/containers/SnippetService/Conditions/FomComponents'
import { useConditionFilterValidation } from '@/containers/SnippetService/hooks'

const DetailForm: FC<Props> = (props) => {
    const { formData, refs, resetIndex, setFilterError } = props
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

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const invalidFilters = useConditionFilterValidation({ watch })

    useEffect(() => {
        isFunction(setFilterError) && setFilterError(invalidFilters)
    }, [invalidFilters, setFilterError])

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
                            attribute: _(g.enabled),
                            value: (
                                <FormGroup error={get(errors, 'enabled.message')} id='enabled'>
                                    <div>
                                        <Controller
                                            control={control}
                                            name='enabled'
                                            render={({ field: { onChange, value } }) => (
                                                <Switch
                                                    checked={value}
                                                    onChange={(e) => {
                                                        onChange(e)
                                                        updateField('enabled', e.target.checked)
                                                    }}
                                                    style={{
                                                        position: 'relative',
                                                        top: '2px',
                                                    }}
                                                />
                                            )}
                                        />
                                    </div>
                                </FormGroup>
                            ),
                        },
                        {
                            attribute: _(confT.configuration),
                            value: (
                                <Tag
                                    onClick={() =>
                                        navigate(
                                            generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, {
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
                            value: formData.timestamp ? (
                                <FormInput disabled onChange={() => {}} value={formatText(formData.timestamp, formatDate, formatTime)} />
                            ) : (
                                '-'
                            ),
                        },
                        {
                            attribute: _(g.version),
                            value: <FormInput disabled onChange={() => {}} value={formData.version || ''} />,
                        },
                    ]}
                />
            </div>
            <Spacer type='pt-8'>
                <Spacer type='mb-6'>
                    <Headline type='h5'>{_(g.filters)}</Headline>
                    <p style={{ margin: '4px 0 0 0' }}>Short description...</p>

                    <Step2FormComponent
                        isActivePage={true}
                        refs={pick(refs, ['filterDeviceId', 'filterJqExpression', 'filterResourceHref', 'filterResourceType'])}
                        setValue={setValue}
                        updateField={updateField}
                        watch={watch}
                    />

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
