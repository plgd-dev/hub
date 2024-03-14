import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'
import Switch from '@shared-ui/components/Atomic/Switch'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

import { Props, Inputs } from './DetailForm.types'
import { messages as g } from '../../../Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { useLinkedHubsList } from '@/containers/DeviceProvisioning/hooks'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultFormData } = props

    console.log(defaultFormData)

    const { updateData, setFormError, commonInputProps, commonFormGroupProps } = useContext(FormContext)
    const { data: hubsData } = useLinkedHubsList()

    const {
        formState: { errors },
        register,
        watch,
        control,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        errorKey: 'tab1',
    })

    const linkedHubs = useMemo(() => {
        if (hubsData) {
            return hubsData.map((linkedHub: { name: string; id: string }) => ({
                value: linkedHub.id,
                label: linkedHub.name,
            }))
        }

        return []
    }, [hubsData])

    const hubIds = watch('hubIds')

    console.log(hubIds)

    const topRows = useMemo(
        () => [
            {
                attribute: _(g.name),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                        <FormInput {...commonInputProps} placeholder={_(g.name)} {...register('name', { required: true, validate: (val) => val !== '' })} />
                    </FormGroup>
                ),
            },
            {
                attribute: _(t.linkedHubs),
                value: (
                    <FormGroup
                        {...commonFormGroupProps}
                        error={errors.hubIds ? _(g.requiredField, { field: _(t.linkedHubs) }) : undefined}
                        id='matchingCertificate'
                    >
                        <div>
                            <Controller
                                control={control}
                                name='hubIds'
                                render={({ field: { onChange, value } }) => (
                                    <FormSelect
                                        inlineStyle
                                        isMulti
                                        align='right'
                                        error={!!errors.name}
                                        onChange={(options: OptionType[]) => onChange(options.map((option) => option.value))}
                                        options={linkedHubs}
                                        size='small'
                                    />
                                )}
                            />
                        </div>
                    </FormGroup>
                ),
            },
            { attribute: _(g.ownerID), value: defaultFormData?.owner },
        ],
        [commonFormGroupProps, commonInputProps, defaultFormData?.owner, errors.name, register]
    )

    const bottomRows = [
        {
            attribute: _(g.certificate),
            value: (
                <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='matchingCertificate'>
                    <div>
                        <FormSelect
                            inlineStyle
                            align='right'
                            error={!!errors.name}
                            options={[
                                { value: '1', label: 'Opt1' },
                                { value: '2', label: 'Opt2' },
                            ]}
                        />
                    </div>
                </FormGroup>
            ),
        },
        {
            attribute: _(t.matchingCertificate),
            value: (
                <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='matchingCertificate'>
                    <div>
                        <FormSelect
                            inlineStyle
                            align='right'
                            error={!!errors.name}
                            options={[
                                { value: '1', label: 'Opt1' },
                                { value: '2', label: 'Opt2' },
                            ]}
                        />
                    </div>
                </FormGroup>
            ),
        },
        {
            attribute: _(t.enableExpiredCertificates),
            value: (
                <Spacer type='pr-4'>
                    <Switch size='small' />
                </Spacer>
            ),
        },
    ]

    return (
        <div>
            <form>
                <Spacer type='mb-4'>
                    <Headline type='h6'>{_(t.enrollmentConfiguration)}</Headline>
                </Spacer>
                <SimpleStripTable leftColSize={6} rightColSize={6} rows={topRows} />
                <Spacer type='mt-8 mb-4'>
                    <Headline type='h6'>{_(t.deviceAuthentication)}</Headline>
                </Spacer>
                <SimpleStripTable rows={bottomRows} />
            </form>
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
