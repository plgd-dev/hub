import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@plgd/shared-ui/src/components/Atomic/TileToggle'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import * as commonStyles from '../../LinkNewHubPage.styles'
import { Props, Inputs } from './Step3.types'
import SubStepTls from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/Steps/SubStepTls'
import SubStepButtons from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/Steps/SubStepButtons/SubStepButtons'

const Step3: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError, setStep } = useContext(FormContext)

    const {
        formState: { errors },
        register,
        control,
        watch,
        setValue,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step3' })

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.certificateAuthority)}</h1>
            <p css={[commonStyles.description, commonStyles.descriptionLarge]}>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
            </p>

            <h2 css={commonStyles.subHeadline}>{_(t.generalKeepAlive)}</h2>
            <p css={commonStyles.description}>Short description...</p>

            <h3 css={commonStyles.groupHeadline}>{_(t.general)}</h3>
            <FormGroup
                error={errors.certificateAuthority?.grpc?.address ? _(g.requiredField, { field: _(t.address) }) : undefined}
                id='certificateAuthority.grpc.address'
            >
                <FormLabel text={_(t.address)} />
                <FormInput
                    {...register('certificateAuthority.grpc.address', {
                        required: true,
                        validate: (val) => val !== '',
                    })}
                />
            </FormGroup>

            <h3 css={commonStyles.groupHeadline}>{_(t.keepAlive)}</h3>

            <Controller
                control={control}
                name='certificateAuthority.grpc.keepAlive.time'
                render={({ field: { onChange, value } }) => (
                    <TimeoutControl
                        watchUnitChange
                        align='left'
                        defaultTtlValue={parseInt(value, 10)}
                        defaultValue={parseInt(value, 10)}
                        i18n={{
                            default: _(g.default),
                            duration: _(g.time),
                            unit: _(g.metric),
                            placeholder: _(g.placeholder),
                        }}
                        onChange={(v) => onChange(v.toString())}
                        rightStyle={{
                            width: 150,
                        }}
                    />
                )}
            />

            <Spacer type='pt-5'>
                <Controller
                    control={control}
                    name='certificateAuthority.grpc.keepAlive.timeout'
                    render={({ field: { onChange, value } }) => (
                        <TimeoutControl
                            watchUnitChange
                            align='left'
                            defaultTtlValue={parseInt(value, 10)}
                            defaultValue={parseInt(value, 10)}
                            i18n={{
                                default: _(g.default),
                                duration: _(g.timeout),
                                unit: _(g.metric),
                                placeholder: _(g.placeholder),
                            }}
                            onChange={(v) => onChange(v.toString())}
                            rightStyle={{
                                width: 150,
                            }}
                        />
                    )}
                    rules={{
                        required: true,
                    }}
                />
            </Spacer>

            <Spacer type='pt-5'>
                <Controller
                    control={control}
                    name='certificateAuthority.grpc.keepAlive.permitWithoutStream'
                    render={({ field: { onChange, value } }) => (
                        <TileToggle checked={(value as boolean) ?? false} name={_(t.permitWithoutStream)} onChange={onChange} />
                    )}
                />
            </Spacer>

            <SubStepTls control={control} prefix='certificateAuthority.grpc.' setValue={setValue} watch={watch} />

            <SubStepButtons onClickBack={() => setStep?.(1)} onClickNext={() => setStep?.(3)} />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
