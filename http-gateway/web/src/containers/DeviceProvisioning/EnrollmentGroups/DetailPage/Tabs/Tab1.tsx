import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'

import { Props, Inputs } from './Tab1.types'
import { messages as g } from '../../../../Global.i18n'
import { messages as t } from '../../EnrollmentGroups.i18n'
import Switch from '@plgd/shared-ui/src/components/Atomic/Switch'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import Tooltip, { tooltipVariants } from '@plgd/shared-ui/src/components/Atomic/Tooltip'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultFormData, hubData } = props

    const { updateData, setFormError, commonInputProps, commonFormGroupProps } = useContext(FormContext)

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

    const topRows = useMemo(() => {
        const rows: Row[] = [
            {
                attribute: _(g.name),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                        <FormInput {...commonInputProps} placeholder={_(g.name)} {...register('name', { required: true, validate: (val) => val !== '' })} />
                    </FormGroup>
                ),
            },
        ]

        if (hubData?.name) {
            rows.push({ attribute: _(g.linkedHub), value: hubData?.name })
        }

        rows.push({ attribute: _(g.ownerID), value: defaultFormData?.owner })

        return rows
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

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

    console.log('---')
    console.log(defaultFormData)
    console.log(hubData)

    return (
        <div>
            <form>
                <SimpleStripTable rows={topRows} />
                <Spacer type='mt-8 mb-4'>
                    <Headline type='h6'>{_(t.deviceAuthentication)}</Headline>
                </Spacer>
                <SimpleStripTable rows={bottomRows} />
            </form>

            <br />
            <br />
            <br />
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
