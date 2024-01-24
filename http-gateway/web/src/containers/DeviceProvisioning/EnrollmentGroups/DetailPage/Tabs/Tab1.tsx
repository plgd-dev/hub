import React, { FC, useContext, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { SubmitHandler, useForm } from 'react-hook-form'
import ReactDOM from 'react-dom'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { selectAligns } from '@shared-ui/components/Atomic'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import { useIsMounted } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'

import { Props, Inputs } from './Tab1.types'
import { messages as g } from '../../../../Global.i18n'
import { messages as t } from '../../EnrollmentGroups.i18n'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { data, hubData } = props

    const isMounted = useIsMounted()
    const { collapsed } = useContext(AppContext)

    const {
        handleSubmit,
        formState: { errors, isDirty, dirtyFields },
        reset,
        register,
        getValues,
    } = useForm<Inputs>({
        mode: 'all',
        reValidateMode: 'onSubmit',
        values: {
            name: data?.name ?? '',
        },
    })

    const topRows = useMemo(() => {
        const rows: Row[] = [
            {
                attribute: _(g.name),
                value: (
                    <FormGroup errorTooltip fullSize error={errors.name ? _(t.nameError) : undefined} id='name' marginBottom={false}>
                        <FormInput inlineStyle align={selectAligns.RIGHT} placeholder={_(g.name)} {...register('name', { validate: (val) => val !== '' })} />
                    </FormGroup>
                ),
            },
        ]

        if (hubData?.name) {
            rows.push({ attribute: _(g.linkedHub), value: hubData?.name })
        }

        rows.push({ attribute: _(g.ownerID), value: data?.owner })

        return rows
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data?.name, data?.owner, hubData?.name])

    const bottomRows = useMemo(
        () => [
            { attribute: _(g.certificate), value: 'TODO' },
            { attribute: _(t.matchingCertificate), value: 'TODO' },
            { attribute: _(t.enableExpiredCertificates), value: 'TODO' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const onSubmit: SubmitHandler<Inputs> = (data) => {
        console.log('submit!')
    }

    return (
        <div>
            <form onSubmit={handleSubmit(onSubmit)}>
                <SimpleStripTable rows={topRows} />
                <Spacer type='mt-8 mb-4'>
                    <Headline type='h6'>{_(t.deviceAuthentication)}</Headline>
                </Spacer>
                <SimpleStripTable rows={bottomRows} />
            </form>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button disabled={Object.keys(errors).length > 0} onClick={() => onSubmit(getValues())} variant='primary'>
                                {_(g.saveChanges)}
                            </Button>
                        }
                        actionSecondary={
                            <Button onClick={() => reset()} variant='secondary'>
                                {_(g.reset)}
                            </Button>
                        }
                        attribute={_(g.changesMade)}
                        leftPanelCollapsed={collapsed}
                        show={isDirty}
                        value={`${Object.keys(dirtyFields).length} ${Object.keys(dirtyFields).length > 1 ? _(t.fields) : _(t.field)}`}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
