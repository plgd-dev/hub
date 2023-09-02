import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
    remoteUiClient: {
        id: 'remoteClients.remoteUiClient',
        defaultMessage: 'Remote UI Client',
    },
    remoteClient: {
        id: 'remoteClients.remoteClient',
        defaultMessage: 'Remote Client',
    },
    remoteClients: {
        id: 'remoteClients.remoteClients',
        defaultMessage: 'Remote Clients',
    },
    recentCommands: {
        id: 'remoteClients.recentCommands',
        defaultMessage: 'Recent Commands',
    },
    client: {
        id: 'remoteClients.client',
        defaultMessage: 'Client',
    },
    clientName: {
        id: 'remoteClients.deviceName',
        defaultMessage: 'Client name',
    },
    deviceAuthenticationMode: {
        id: 'remoteClients.deviceAuthenticationMode',
        defaultMessage: 'Device Authentication Mode',
    },
    subjectId: {
        id: 'remoteClients.subjectId',
        defaultMessage: 'Subject ID',
    },
    key: {
        id: 'remoteClients.key',
        defaultMessage: 'Key',
    },
    config: {
        id: 'remoteClients.config',
        defaultMessage: 'Config',
    },
    clientNameError: {
        id: 'remoteClients.clientNameError',
        defaultMessage: 'Client name error message',
    },
    addNewClient: {
        id: 'remoteClients.addNewClient',
        defaultMessage: 'Add a new client',
    },
    editClient: {
        id: 'remoteClients.editClient',
        defaultMessage: 'Edit client',
    },
    preSharedSubjectIdError: {
        id: 'remoteClients.preSharedSubjectIdError',
        defaultMessage: 'SharedSubjectIdError error message',
    },
    preSharedKeyError: {
        id: 'remoteClients.preSharedKeyError',
        defaultMessage: 'preSharedKeyError error message',
    },
    clientUrl: {
        id: 'remoteClients.clientUrl',
        defaultMessage: 'Client Url',
    },
    clientUrlError: {
        id: 'remoteClients.clientUrlError',
        defaultMessage: 'Client Url error message',
    },
    addClientButton: {
        id: 'remoteClients.addClientButton',
        defaultMessage: 'Add the client',
    },
    addClientButtonLoading: {
        id: 'remoteClients.addClientButtonLoading',
        defaultMessage: 'Checking the client',
    },
    clientInformation: {
        id: 'remoteClients.clientInformation',
        defaultMessage: 'Client information',
    },
    copy: {
        id: 'remoteClients.copy',
        defaultMessage: 'Copy',
    },
    done: {
        id: 'remoteClients.done',
        defaultMessage: 'Done',
    },
    success: {
        id: 'remoteClients.success',
        defaultMessage: 'Success',
    },
    clientSuccess: {
        id: 'remoteClients.clientSuccess',
        defaultMessage: 'The client was found',
    },
    error: {
        id: 'remoteClients.error',
        defaultMessage: 'Error',
    },
    clientError: {
        id: 'remoteClients.clientError',
        defaultMessage:
            'Failed to add the remote client. The certificate for the provided URL may not have been accepted by browser. Please ensure you open the {remoteClientUrl} in your browser, verify and accept the certificate before attempting to add the client.',
    },
    version: {
        id: 'remoteClients.version',
        defaultMessage: 'Version',
    },
    ipAddress: {
        id: 'remoteClients.ipAddress',
        defaultMessage: 'IP Address',
    },
    reachable: {
        id: 'remoteClients.reachable',
        defaultMessage: 'Reachable',
    },
    unReachable: {
        id: 'remoteClients.unReachable',
        defaultMessage: 'Unreachable',
    },
    deleteClientMessage: {
        id: 'remoteClients.deleteClientMessage',
        defaultMessage: 'Are you sure you want to delete this remote UI client?',
    },
    deleteClientsMessage: {
        id: 'remoteClients.deleteClientsMessage',
        defaultMessage: 'Are you sure you want to delete {count} remote UI clients?',
    },
    clientsDeleted: {
        id: 'remoteClients.clientsDeleted',
        defaultMessage: 'remote clients deleted',
    },
    clientsDeletedMessage: {
        id: 'remoteClients.clientsDeletedMessage',
        defaultMessage: 'The selected remote clients were successfully deleted.',
    },
    notFoundRemoteClientMessage: {
        id: 'remoteClients.notFoundRemoteClientMessage',
        defaultMessage: 'The remote client you are looking for does not exist.',
    },
    authError: {
        id: 'remoteClients.authError',
        defaultMessage: 'Authorization server error',
    },
    clientsUpdated: {
        id: 'remoteClients.clientsDeleted',
        defaultMessage: 'remote client updated',
    },
    clientsUpdatedMessage: {
        id: 'remoteClients.clientsDeletedMessage',
        defaultMessage: 'The remote client was successfully updated.',
    },
    initializedByAnotherDesc: {
        id: 'remoteClients.initializedByAnotherDesc',
        defaultMessage:
            'Application Initialization Restricted. Please ensure the remote client user logs out before proceeding. Only after the different user has logged out, will you be able to utilize the application.',
    },
    certificateAcceptDescription: {
        id: 'remoteClients.certificateAcceptDescription',
        defaultMessage:
            'Before adding a remote client, verify their TLS certificate for security. To proceed, open the URL in your browser, verify and accept the certificate. Adding a client involves sharing credentials, so ensure you trust them.',
    },
})
