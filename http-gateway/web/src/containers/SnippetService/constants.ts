export const SnippetServiceApiEndpoints = {
    CONFIGURATIONS: '/snippet-service/api/v1/configurations',
    CONFIGURATIONS_APPLIED: '/snippet-service/api/v1/configurations/applied',
    CONDITIONS: '/snippet-service/api/v1/conditions',
}

// Maximum amount of snippet-service filled into one delete request.
// (if there is more snippetServiceIds then the provided number, it creates more chunks of a maximum of this number)
export const DELETE_CHUNK_SIZE = 50
