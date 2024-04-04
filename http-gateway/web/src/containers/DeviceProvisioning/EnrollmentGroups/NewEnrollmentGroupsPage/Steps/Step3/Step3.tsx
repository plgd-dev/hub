import React, { FC, useContext, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'

import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as t } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.i18n'
import { DetailFromChunk3 } from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailFormChunks'
import { Inputs } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.types'
import { messages as g } from '@/containers/Global.i18n'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/EnrollmentGroups/validationSchema'

const Step3: FC<any> = (props) => {
    const { defaultFormData, onSubmit } = props

    const { formatMessage: _ } = useIntl()

    const { setStep } = useContext(FormContext)
    const schema = useValidationsSchema('group3')

    const {
        formState: { errors, isValid },
        register,
        updateField,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'step3',
        schema,
    })

    const [preSharedKeySettings, setPreSharedKeySettings] = useState(false)
    const preSharedKey = watch('preSharedKey')

    useEffect(() => {
        setPreSharedKeySettings(!!preSharedKey)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    return (
        <form>
            <FullPageWizard.Headline>{_(t.deviceCredentials)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(t.addEnrollmentGroupDeviceCredentialsDescription)}</FullPageWizard.Description>

            <DetailFromChunk3
                errors={errors}
                preSharedKeySettings={preSharedKeySettings}
                register={register}
                setPreSharedKeySettings={setPreSharedKeySettings}
                setValue={setValue}
                updateField={updateField}
            />

            <StepButtons
                disableNext={preSharedKeySettings && !isValid}
                i18n={{
                    back: _(g.back),
                    continue: _(g.create),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(1)}
                onClickNext={() => onSubmit?.()}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
