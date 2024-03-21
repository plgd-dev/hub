import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'

import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'
import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'

import { messages as t } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.i18n'
import { DetailFromChunk3 } from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailFormChunks'
import { Inputs } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.types'
import { messages as g } from '@/containers/Global.i18n'

const Step3: FC<any> = (props) => {
    const { defaultFormData, onSubmit } = props

    const { formatMessage: _ } = useIntl()

    const { updateData, setFormDirty, setFormError, setStep } = useContext(FormContext)
    const {
        formState: { errors },
        register,
        updateField,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        setFormDirty,
        errorKey: 'step3',
    })

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.deviceCredentials)}</h1>
            <p css={[commonStyles.description, commonStyles.descriptionLarge]}>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
            </p>

            <DetailFromChunk3 errors={errors} register={register} setValue={setValue} updateField={updateField} watch={watch} />

            <StepButtons
                i18n={{
                    back: _(g.back),
                    continue: _(g.create),
                }}
                onClickBack={() => setStep?.(2)}
                onClickNext={() => onSubmit?.()}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
