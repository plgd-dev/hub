import React, { FC, useCallback, useContext, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { useParams } from 'react-router-dom'
import { useForm } from '@shared-ui/common/hooks'
import { Controller, useFieldArray } from 'react-hook-form'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'
import Button from '@shared-ui/components/Atomic/Button'
import IconClose from '@shared-ui/components/Atomic/Icon/components/IconClose'
import { convertSize } from '@shared-ui/components/Atomic'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import ValidationMessage from '@shared-ui/components/Atomic/ValidationMessage'

import { Props, Inputs } from './Tab1.types'
import { messages as g } from '../../../../../Global.i18n'
import { messages as t } from '../../../LinkedHubs.i18n'
import * as styles from '../Tab.styles'
import isFunction from 'lodash/isFunction'

const Tab1: FC<Props> = (props) => {
    const { defaultFormData, resetIndex } = props
    const { formatMessage: _ } = useIntl()
    const { hubId } = useParams()

    const { updateData, setFormError, setFormDirty, commonFormGroupProps, commonInputProps } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        updateField,
        control,
        reset,
        watch,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, setFormDirty, errorKey: 'tab1' })

    const { fields, append, remove } = useFieldArray({
        control,
        name: 'gateways',
    })

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const gateways = watch('gateways')

    const checkGateways = useCallback(
        (nextGateways = gateways) => {
            if (isFunction(setFormError)) {
                const hasError = fields.length === 0 || nextGateways.some((f) => f.value === '')
                setFormError((prevState: any) => ({ ...prevState, tab1: hasError }))
            }
        },
        [fields.length, gateways, setFormError]
    )

    return (
        <form>
            <SimpleStripTable
                leftColSize={7}
                rightColSize={5}
                rows={
                    [
                        {
                            attribute: _(g.id),
                            key: 'r-id',
                            value: <FormInput {...commonInputProps} disabled value={hubId} />,
                        },
                        {
                            attribute: _(g.name),
                            key: 'r-name',
                            value: (
                                <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                                    <FormInput
                                        {...commonInputProps}
                                        placeholder={_(g.name)}
                                        {...register('name', { required: true, validate: (val) => val !== '' })}
                                        key='name'
                                        onBlur={(e) => updateField('name', e.target.value)}
                                    />
                                </FormGroup>
                            ),
                        },
                        ...fields?.map((field, index) => ({
                            attribute: _(t.deviceGateway),
                            key: `r-${field.id}`,
                            value: (
                                <FormGroup
                                    {...commonFormGroupProps}
                                    error={errors.gateways?.[index] ? _(t.deviceGateway, { field: _(t.deviceGateway) }) : undefined}
                                    id={`gateways.${index}`}
                                    key={field.id}
                                >
                                    <></>
                                    <Controller
                                        control={control}
                                        name={`gateways.${index}` as any}
                                        render={({ field: { onChange, value } }) => (
                                            <div css={styles.flex}>
                                                <FormInput
                                                    {...commonInputProps}
                                                    onBlur={(e) => updateField(`gateways.${index}`, { value: e.target.value, id: field.id }, true)}
                                                    onChange={(v) => {
                                                        onChange({ value: v.target.value, id: field.id })
                                                        checkGateways()
                                                    }}
                                                    value={value.value}
                                                />
                                                <a
                                                    css={styles.removeIcon}
                                                    href='#'
                                                    onClick={(e) => {
                                                        e.preventDefault()
                                                        e.stopPropagation()
                                                        remove(index)

                                                        const nextGateways = gateways.filter((_, key) => key !== index)

                                                        updateField('gateways', nextGateways)

                                                        checkGateways(nextGateways)
                                                    }}
                                                >
                                                    <IconClose {...convertSize(20)} />
                                                </a>
                                            </div>
                                        )}
                                        rules={{
                                            required: true,
                                            validate: (val) => val !== '',
                                        }}
                                    />
                                </FormGroup>
                            ),
                        })),
                        fields?.length === 0 && {
                            attribute: _(t.deviceGateway),
                            key: 'r-empty',
                            value: (
                                <FormGroup id='gateways' marginBottom={false}>
                                    <ValidationMessage>{_(t.deviceGatewayEmptyError)}</ValidationMessage>
                                </FormGroup>
                            ),
                        },
                        {
                            attribute: '',
                            key: 'add-button',
                            value: (
                                <Button
                                    disabled={gateways ? gateways.some((g) => g.value === '') : true}
                                    icon={<IconPlus />}
                                    onClick={(e) => {
                                        e.preventDefault()
                                        e.stopPropagation()

                                        append({ value: '' })
                                        isFunction(setFormError) && setFormError((prevState: any) => ({ ...prevState, tab1: true }))
                                    }}
                                    size='small'
                                    variant='filter'
                                >
                                    {_(t.addDeviceGateway)}
                                </Button>
                            ),
                        },
                    ].filter((r) => !!r) as Row[]
                }
            />
        </form>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
