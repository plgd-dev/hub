import React, { FC, useContext, useState } from 'react'
import { useIntl } from 'react-intl'

import Column from '@shared-ui/components/Atomic/Grid/Column'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import Headline from '@shared-ui/components/Atomic/Headline'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import Alert, { severities } from '@shared-ui/components/Atomic/Alert'
import Button from '@shared-ui/components/Atomic/Button'
import IconClose from '@shared-ui/components/Atomic/Icon/components/IconClose'
import { convertSize } from '@shared-ui/components/Atomic'
import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import Checkbox from '@shared-ui/components/Atomic/Checkbox'
import TooltipIcon from '@shared-ui/components/Atomic/Tooltip/TooltipIcon'
import LinkExpander from '@shared-ui/components/Atomic/LinkExpander/LinkExpander'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { Inputs, Props } from './Step2.types'
import * as styles from './Step2.styles'
import { flexCenter, verticalSeparator } from './Step2.styles'

const options = [
    { value: 'chocolate', label: 'Chocolate' },
    { value: 'strawberry', label: 'Strawberry' },
    { value: 'vanilla', label: 'Vanilla' },
]

const Step2: FC<Props> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { updateData, setFormError } = useContext(FormContext)
    const [newCertificateView, setNewCertificateView] = useState(false)
    const [advancedConfiguration, setAdvancedConfiguration] = useState(false)

    const {
        formState: { errors },
        register,
        watch,
        setValue,
    } = useForm<Inputs>({ defaultFormData, updateData, setFormError, errorKey: 'step2' })

    const certificate = watch('certificate')
    const caPool = watch('caPool')

    return (
        <Spacer type='py-6 px-10'>
            <form>
                <Row gutters={false}>
                    {!newCertificateView && <Column lg={3} size={3} sm={2}></Column>}
                    <Column size={6}>
                        <Alert noSeverityBg css={styles.alertPadding} severity={severities.SUCCESS}>
                            Connection to plgd hub instance confirmed. Please verify retrieved information and make any necessary changes.
                        </Alert>

                        <Spacer type='mb-4'>
                            <Headline type='h6'>Hub information</Headline>
                        </Spacer>
                        <Row>
                            <Column size={6}>
                                <FormGroup error={errors.hubName ? _(g.requiredField, { field: _(t.hubName) }) : undefined} id='hubName'>
                                    <FormLabel text={_(t.hubName)} />
                                    <FormInput
                                        {...register('hubName', {
                                            required: true,
                                            validate: (val) => val !== '',
                                        })}
                                    />
                                </FormGroup>
                            </Column>
                            <Column size={6}>
                                <FormGroup error={errors.endpoint ? _(g.requiredField, { field: _(t.endpoint) }) : undefined} id='endpoint'>
                                    <FormLabel text={_(t.endpoint)} tooltipMaxWidth={270} tooltipText={_(t.endpointDescription)} />
                                    <FormInput
                                        {...register('endpoint', {
                                            required: true,
                                            validate: (val) => val !== '',
                                        })}
                                    />
                                </FormGroup>
                            </Column>
                        </Row>
                        <FormGroup error={errors.certificate ? _(g.requiredField, { field: _(t.certificate) }) : undefined} id='certificate'>
                            <FormLabel text={_(t.certificate)} tooltipMaxWidth={270} tooltipText={_(t.certificateDescription)} />
                            <FormSelect
                                footerLinks={[
                                    {
                                        title: 'Register a new certificate',
                                        onClick: () => {
                                            setNewCertificateView(true)
                                        },
                                    },
                                ]}
                                name='certificate'
                                onChange={(option) => {
                                    if (option?.value && newCertificateView) {
                                        setNewCertificateView(false)
                                    }
                                    if (option?.value !== certificate) {
                                        setValue('certificate', option?.value)
                                    }
                                }}
                                options={options}
                                value={options.find((o) => o.value === certificate)}
                            />
                        </FormGroup>
                        {!!certificate && (
                            <Spacer type='pt-5'>
                                <Spacer type='mb-6'>
                                    <Headline type='h6'>{_(t.certificateAuthorityConfiguration)}</Headline>
                                </Spacer>
                                <div css={styles.flex}>
                                    <Checkbox defaultChecked={false} label={_(t.disableIssuingIdentityCertificate)} name='disableIssuingIdentityCertificate' />
                                    <TooltipIcon content={_(t.disableIssuingIdentityCertificateDescription)} />
                                </div>
                                <Spacer type='pt-6'>
                                    <FormGroup
                                        error={errors.certificateAuthorityAddress ? _(g.requiredField, { field: _(t.certificateAuthorityAddress) }) : undefined}
                                        id='certificateAuthorityAddress'
                                    >
                                        <FormLabel
                                            text={_(t.certificateAuthorityAddress)}
                                            tooltipMaxWidth={270}
                                            tooltipText={_(t.certificateAuthorityAddressDescription)}
                                        />
                                        <FormInput
                                            {...register('certificateAuthorityAddress', {
                                                required: true,
                                                validate: (val) => val !== '',
                                            })}
                                        />
                                    </FormGroup>
                                </Spacer>
                                <Spacer type='pt-2'>
                                    <LinkExpander
                                        i18n={{
                                            show: _(g.show),
                                            hide: _(g.hide),
                                            name: _(t.advancedConfiguration),
                                        }}
                                        show={advancedConfiguration}
                                        toggleView={() => setAdvancedConfiguration(!advancedConfiguration)}
                                    >
                                        <Spacer type='mb-4'>
                                            <Headline type='h6'>{_(t.certificateAuthorityConfiguration)}</Headline>
                                        </Spacer>
                                        <FormGroup error={errors.caPool ? _(g.requiredField, { field: _(t.caPool) }) : undefined} id='caPool'>
                                            <FormLabel text={_(t.caPool)} />
                                            <FormSelect
                                                name='caPool'
                                                onChange={(option) => {
                                                    if (option?.value !== certificate) {
                                                        setValue('caPool', option?.value)
                                                    }
                                                }}
                                                options={options}
                                                value={options.find((o) => o.value === certificate)}
                                            />
                                        </FormGroup>
                                        <FormGroup
                                            error={errors.clientCertificate ? _(g.requiredField, { field: _(t.clientCertificate) }) : undefined}
                                            id='clientCertificate'
                                        >
                                            <FormLabel text={_(t.clientCertificate)} />
                                            <FormSelect
                                                name='clientCertificate'
                                                onChange={(option) => {
                                                    if (option?.value !== certificate) {
                                                        setValue('clientCertificate', option?.value)
                                                    }
                                                }}
                                                options={options}
                                                value={options.find((o) => o.value === certificate)}
                                            />
                                        </FormGroup>
                                        <FormGroup
                                            error={
                                                errors.clientCertificatePrivateKey ? _(g.requiredField, { field: _(t.clientCertificatePrivateKey) }) : undefined
                                            }
                                            id='clientCertificate'
                                        >
                                            <FormLabel text={_(t.clientCertificatePrivateKey)} />
                                            <FormSelect
                                                menuPortalTarget={document.getElementById('modal-root')}
                                                menuZIndex={5}
                                                name='clientCertificatePrivateKey'
                                                onChange={(option) => {
                                                    if (option?.value !== certificate) {
                                                        setValue('clientCertificatePrivateKey', option?.value)
                                                    }
                                                }}
                                                options={options}
                                                value={options.find((o) => o.value === certificate)}
                                            />
                                        </FormGroup>
                                        <Row>
                                            <Column size={6}>
                                                <FormGroup
                                                    error={errors.keepAliveTime ? _(g.requiredField, { field: _(t.keepAliveTime) }) : undefined}
                                                    id='keepAliveTime'
                                                >
                                                    <FormLabel text={_(t.keepAliveTime)} />
                                                    <FormInput {...register('keepAliveTime', { valueAsNumber: true })} rightContent='s' />
                                                </FormGroup>
                                            </Column>
                                            <Column size={6}>
                                                <FormGroup
                                                    error={errors.keepAliveTimeout ? _(g.requiredField, { field: _(t.keepAliveTimeout) }) : undefined}
                                                    id='keepAliveTimeout'
                                                >
                                                    <FormLabel text={_(t.keepAliveTimeout)} />
                                                    <FormInput {...register('keepAliveTimeout', { valueAsNumber: true })} rightContent='s' />
                                                </FormGroup>
                                            </Column>
                                        </Row>
                                    </LinkExpander>
                                </Spacer>
                                <Spacer type='pt-8'>
                                    <Button variant='primary'>{_(g.finish)}</Button>
                                </Spacer>
                            </Spacer>
                        )}
                    </Column>
                    {newCertificateView && (
                        <>
                            <Column css={styles.flexCenter} size={1}>
                                <div css={styles.verticalSeparator} />
                            </Column>
                            <Column size={5}>
                                <Spacer type='mb-4'>
                                    <div css={styles.flex}>
                                        <Headline type='h6'>{_(t.registerNewCertificate)}</Headline>
                                        <a
                                            css={styles.close}
                                            href='#'
                                            onClick={(e) => {
                                                e.preventDefault()
                                                setNewCertificateView(false)
                                            }}
                                        >
                                            <IconClose {...convertSize(20)} />
                                        </a>
                                    </div>
                                </Spacer>
                                <FormGroup
                                    error={errors.certificateName ? _(g.requiredField, { field: _(t.certificateName) }) : undefined}
                                    id='certificateName'
                                >
                                    <FormLabel text={_(t.certificateName)} />
                                    <FormInput
                                        {...register('certificateName', {
                                            required: true,
                                            validate: (val) => val !== '',
                                        })}
                                    />
                                </FormGroup>
                                <Dropzone
                                    // accept={{
                                    //     cert: ['.pem', '.cer'],
                                    // }}
                                    smallPadding
                                    customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                                    description={_(t.uploadCertDescription)}
                                    maxFiles={1}
                                    title={_(t.uploadCertTitle)}
                                />
                                <Spacer type='mt-8'>
                                    <Button variant='primary'>{_(t.saveCertificate)}</Button>
                                </Spacer>
                            </Column>
                        </>
                    )}
                </Row>
            </form>
        </Spacer>
    )
}

Step2.displayName = 'Step2'

export default Step2
