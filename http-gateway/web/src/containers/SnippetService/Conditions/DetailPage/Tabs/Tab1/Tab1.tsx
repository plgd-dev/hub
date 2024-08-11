import React, { FC, useEffect } from 'react'
import { generatePath, useNavigate } from 'react-router-dom'
import { useIntl } from 'react-intl'
import get from 'lodash/get'
import { Controller } from 'react-hook-form'

import { messages as g } from '@/containers/Global.i18n'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import Switch from '@shared-ui/components/Atomic/Switch'
import { useForm } from '@shared-ui/common/hooks'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Tag from '@shared-ui/components/Atomic/Tag'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { pages } from '@/routes'
import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { formatText } from '@/containers/PendingCommands/DateFormat'
import { useValidationsSchema } from '@/containers/SnippetService/Conditions/DetailPage/validationSchema'
import { Props, Inputs } from './Tab1.types'
import testId from '@/testId'

const Tab1: FC<Props> = (props) => {
    const { defaultFormData, resetIndex } = props

    const { formatMessage: _, formatDate, formatTime } = useIntl()
    const navigate = useNavigate()
    const schema = useValidationsSchema('tab1')

    const {
        formState: { errors },
        register,
        updateField,
        control,
        reset,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab1',
        schema,
    })

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    return (
        <SimpleStripTable
            leftColSize={6}
            rightColSize={6}
            rows={[
                {
                    attribute: _(g.name),
                    value: (
                        <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                            <FormInput
                                dataTestId={testId.snippetService.conditions.detail.tab1.form.name}
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
                                        configurationId: defaultFormData.configurationId,
                                        tab: '',
                                    })
                                )
                            }
                            variant={tagVariants.BLUE}
                        >
                            <IconLink />
                            <Spacer type='ml-2'>{defaultFormData.configurationName}</Spacer>
                        </Tag>
                    ),
                },
                {
                    attribute: _(g.lastModified),
                    value: defaultFormData.timestamp ? (
                        <FormInput disabled onChange={() => {}} value={formatText(defaultFormData.timestamp, formatDate, formatTime)} />
                    ) : (
                        '-'
                    ),
                },
                {
                    attribute: _(g.version),
                    value: <FormInput disabled onChange={() => {}} value={defaultFormData.version || ''} />,
                },
            ]}
        />
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
