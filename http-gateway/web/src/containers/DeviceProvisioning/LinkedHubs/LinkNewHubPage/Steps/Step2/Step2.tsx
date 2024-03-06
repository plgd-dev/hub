import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller, useFieldArray } from 'react-hook-form'

import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'
import Button from '@shared-ui/components/Atomic/Button'
import IconClose from '@shared-ui/components/Atomic/Icon/components/IconClose'
import { convertSize } from '@shared-ui/components/Atomic'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Inputs, Props } from './Step2.types'
import * as styles from './Step2.styles'
import * as commonStyles from '../../LinkNewHubPage.styles'
import SubStepButtons from '../SubStepButtons'

const Step2: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        control,
        updateField,
        watch,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step2' })

    const { fields, append, remove } = useFieldArray({
        control,
        name: 'gateways',
    })

    const gateways = watch('gateways')

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.hubDetails)}</h1>
            <p css={[commonStyles.description, commonStyles.descriptionLarge]}>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
            </p>

            <FormGroup error={errors.hubId ? _(g.requiredField, { field: _(g.hubId) }) : undefined} id='hubID'>
                <FormLabel text={_(g.hubId)} />
                <FormInput
                    {...register('hubId', {
                        required: true,
                        validate: (val) => val !== '',
                    })}
                    onBlur={(e) => updateField('hubId', e.target.value)}
                />
            </FormGroup>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormLabel text={_(g.name)} />
                <FormInput
                    {...register('name', {
                        required: true,
                        validate: (val) => val !== '',
                    })}
                    onBlur={(e) => updateField('name', e.target.value)}
                />
            </FormGroup>

            {fields?.map((field, index) => (
                <FormGroup
                    error={errors.gateways?.[index] ? _(t.deviceGateway, { field: _(t.deviceGateway) }) : undefined}
                    id={`gateways.${index}`}
                    key={field.id}
                >
                    <FormLabel text={_(t.deviceGateway)} />
                    <Controller
                        control={control}
                        name={`gateways.${index}` as any}
                        render={({ field: { onChange, value } }) => (
                            <div css={styles.flex}>
                                <FormInput
                                    onBlur={(e) => updateField(`gateways.${index}`, e.target.value)}
                                    onChange={(v) => {
                                        onChange({ value: v.target.value })
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

                                        updateField(
                                            'gateways',
                                            gateways.filter((_, key) => key !== index)
                                        )
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
            ))}

            <Button
                disabled={defaultFormData.gateways[defaultFormData.gateways.length - 1]?.value === ''}
                icon={<IconPlus />}
                onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()

                    append({ value: '' })
                }}
                size='small'
                variant='filter'
            >
                {_(t.addDeviceGateway)}
            </Button>

            <SubStepButtons onClickBack={() => setStep?.(0)} onClickNext={() => setStep?.(2)} />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2
