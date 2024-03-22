import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
    enrollmentGroup: {
        id: 'enrollmentGroups.enrollmentGroup',
        defaultMessage: 'Enrollment Group',
    },
    deleteEnrollmentGroupMessage: {
        id: 'enrollmentGroups.deleteEnrollmentGroupMessage',
        defaultMessage: 'Are you sure you want to delete this enrollment group?',
    },
    deleteEnrollmentGroupsMessage: {
        id: 'enrollmentGroups.deleteRecordsMessage',
        defaultMessage: 'Are you sure you want to delete {count} enrollment groups?',
    },
    deleteEnrollmentGroupsSubTitle: {
        id: 'enrollmentGroups.deleteEnrollmentGroupsSubTitle',
        defaultMessage: 'This action cannot be undone.',
    },
    enrollmentGroupsError: {
        id: 'enrollmentGroups.enrollmentGroupsError',
        defaultMessage: 'Enrollment Groups Error',
    },
    addEnrollmentGroup: {
        id: 'enrollmentGroups.addEnrollmentGroup',
        defaultMessage: 'Add Enrollment Group',
    },
    enrollmentConfiguration: {
        id: 'enrollmentGroups.enrollmentConfiguration',
        defaultMessage: 'Enrollment Configuration',
    },
    deviceCredentials: {
        id: 'enrollmentGroups.deviceCredentials',
        defaultMessage: 'Device Credentials',
    },
    deleteEnrollmentGroupTitle: {
        id: 'enrollmentGroups.deleteProvisioningRecordTitle',
        defaultMessage: 'Delete Provisioning Record',
    },
    deviceAuthentication: {
        id: 'enrollmentGroups.deviceAuthentication',
        defaultMessage: 'Device authentication',
    },
    leadCertificate: {
        id: 'enrollmentGroups.leadCertificate',
        defaultMessage: 'Lead Certificate',
    },
    enableExpiredCertificates: {
        id: 'enrollmentGroups.enableExpiredCertificates',
        defaultMessage: 'Enable Expired Certificates',
    },
    nameError: {
        id: 'enrollmentGroups.nameError',
        defaultMessage: 'Name error',
    },
    fields: {
        id: 'enrollmentGroups.fields',
        defaultMessage: 'fields',
    },
    field: {
        id: 'enrollmentGroups.field',
        defaultMessage: 'field',
    },
    linkedHubs: {
        id: 'enrollmentGroups.linkedHubs',
        defaultMessage: 'Linked Hubs',
    },
    uploadCertDescription: {
        id: 'enrollmentGroups.uploadCertDescription',
        defaultMessage: 'Supported formats: PEM, CRT or CER (max 1 MB)',
    },
    uploadCertTitle: {
        id: 'enrollmentGroups.uploadCertTitle',
        defaultMessage: 'Drag & Drop or Choose file to upload Certificate',
    },
    certificationParsingError: {
        id: 'enrollmentGroups.certificationParsingError',
        defaultMessage: 'Certification Parsing Error',
    },
    certificateDetail: {
        id: 'enrollmentGroups.certificateDetail',
        defaultMessage: 'Certificate Detail',
    },
    preSharedKeySettings: {
        id: 'enrollmentGroups.preSharedKeySettings',
        defaultMessage: 'Pre-Shared key settings',
    },
    preSharedKey: {
        id: 'enrollmentGroups.preSharedKey',
        defaultMessage: 'Pre-Shared key',
    },
    addEnrollmentGroupDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDescription',
        defaultMessage:
            'The new enrollment group establishes parameters such as owner identification, attestation details, and configuration settings for provisioned devices.',
    },
    addEnrollmentGroupDeviceAuthenticationDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDeviceAuthenticationDescription',
        defaultMessage:
            'By configuring the attestation certificate chain, the Device Provisioning Service identifies the enrollment group to which a device belongs by setting the lead certificate.',
    },
    addEnrollmentGroupDeviceCredentialsDescription: {
        id: 'enrollmentGroups.addEnrollmentGroupDeviceCredentialsDescription',
        defaultMessage:
            'The credentials enable the configuration of a pre-shared key for the device owner, facilitating device management within a local area network through the plgd/client application.',
    },
})
