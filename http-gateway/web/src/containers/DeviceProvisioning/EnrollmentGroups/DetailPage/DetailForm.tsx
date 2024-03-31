import React, { FC, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'

import { Props } from './DetailForm.types'
import { messages as g } from '../../../Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { useLinkedHubsList } from '@/containers/DeviceProvisioning/hooks'
import { Inputs } from '../EnrollmentGroups.types'
import { DetailFromChunk2, DetailFromChunk3 } from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailFormChunks'
import notificationId from '@/notificationId'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/EnrollmentGroups/validationSchema'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultFormData, resetIndex } = props

    const { updateData, setFormDirty, setFormError, commonInputProps, commonFormGroupProps } = useContext(FormContext)
    const { data: hubsData } = useLinkedHubsList()
    const schema = useValidationsSchema('combine')

    const {
        formState: { errors },
        register,
        control,
        updateField,
        reset,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        setFormDirty,
        errorKey: 'tab1',
        schema,
    })

    const linkedHubs = useMemo(
        () =>
            hubsData
                ? hubsData.map((linkedHub: { name: string; id: string }) => ({
                      value: linkedHub.id,
                      label: linkedHub.name,
                  }))
                : [],
        [hubsData]
    )

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const [preSharedKeySettings, setPreSharedKeySettings] = useState(false)
    const preSharedKey = watch('preSharedKey')

    useEffect(() => {
        setPreSharedKeySettings(!!preSharedKey)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const topRows = useMemo(
        () => [
            {
                attribute: _(g.name),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                        <FormInput
                            {...commonInputProps}
                            {...register('name', { required: true, validate: (val) => val !== '' })}
                            onBlur={(e) => updateField('name', e.target.value)}
                            placeholder={_(g.name)}
                        />
                    </FormGroup>
                ),
            },
            {
                attribute: _(t.linkedHubs),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.hubIds ? _(g.requiredField, { field: _(t.linkedHubs) }) : undefined} id='linkedHubs'>
                        <div>
                            <Controller
                                control={control}
                                name='hubIds'
                                render={({ field: { onChange, value } }) => (
                                    <FormSelect
                                        inlineStyle
                                        isMulti
                                        align='right'
                                        error={!!errors.hubIds}
                                        onChange={(options: OptionType[]) => {
                                            const v = options.map((option) => option.value)
                                            onChange(v)
                                            updateField('hubIds', v)
                                        }}
                                        options={linkedHubs}
                                        size='small'
                                        value={value ? linkedHubs.filter((linkedHub: { value: string }) => value.includes(linkedHub.value)) : []}
                                    />
                                )}
                            />
                        </div>
                    </FormGroup>
                ),
            },
            {
                attribute: _(g.ownerID),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.owner ? _(g.ownerID, { field: _(g.name) }) : undefined} id='owner'>
                        <FormInput
                            {...commonInputProps}
                            {...register('owner', { required: true, validate: (val) => val !== '' })}
                            onBlur={(e) => updateField('owner', e.target.value)}
                            placeholder={_(g.ownerID)}
                        />
                    </FormGroup>
                ),
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [commonFormGroupProps, commonInputProps, control, errors.hubIds, errors.name, errors.owner, linkedHubs, register, updateField]
    )

    const certificateChain = watch('attestationMechanism.x509.certificateChain')

    return (
        <form>
            <Spacer type='mb-4'>
                <Headline type='h6'>{_(t.enrollmentConfiguration)}</Headline>
            </Spacer>

            <SimpleStripTable leftColSize={6} rightColSize={6} rows={topRows} />

            <Spacer type='mt-8 mb-4'>
                <Headline type='h6'>{_(t.deviceAuthentication)}</Headline>
            </Spacer>

            <DetailFromChunk2
                isEditMode
                certificateChain={certificateChain}
                control={control}
                errorNotificationId={notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR}
                errors={errors}
                setValue={setValue}
                updateField={updateField}
            />

            <Spacer type='mt-8 mb-4'>
                <Headline type='h6'>{_(t.deviceCredentials)}</Headline>
            </Spacer>

            <DetailFromChunk3
                isEditMode
                errors={errors}
                preSharedKeySettings={preSharedKeySettings}
                register={register}
                setPreSharedKeySettings={setPreSharedKeySettings}
                setValue={setValue}
                updateField={updateField}
            />
        </form>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
