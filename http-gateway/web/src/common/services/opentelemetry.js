import { WebTracerProvider } from '@opentelemetry/sdk-trace-web'
import {
  ConsoleSpanExporter,
  SimpleSpanProcessor,
} from '@opentelemetry/sdk-trace-base'
import { registerInstrumentations } from '@opentelemetry/instrumentation'
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch'
import { ZoneContextManager } from '@opentelemetry/context-zone'
import { B3Propagator } from '@opentelemetry/propagator-b3'
import { XMLHttpRequestInstrumentation } from '@opentelemetry/instrumentation-xml-http-request'
import { context, trace } from '@opentelemetry/api'

let webTracer = undefined

const init = (appName = '') => {
  const provider = new WebTracerProvider()
  provider.addSpanProcessor(new SimpleSpanProcessor(new ConsoleSpanExporter()))
  provider.register({
    contextManager: new ZoneContextManager(),
    propagator: new B3Propagator(),
  })

  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        ignoreUrls: [/localhost:3000\/sockjs-node/],
        clearTimingResources: true,
      }),
      new XMLHttpRequestInstrumentation({
        ignoreUrls: [/localhost:3000\/sockjs-node/],
      }),
    ],
  })

  webTracer = provider.getTracer(`${appName}-tracer`)
}

export const withTelemetry = async (restMethod, telemetrySpan) => {
  if (webTracer) {
    const singleSpan = webTracer.startSpan(telemetrySpan)
    let dataToReturn = undefined

    await context.with(
      trace.setSpan(context.active(), singleSpan),
      async () => {
        dataToReturn = await restMethod().then(result => {
          trace
            .getSpan(context.active())
            .addEvent('fetching-single-span-completed')
          singleSpan.end()

          return result
        })
      }
    )

    return dataToReturn
  } else {
    return restMethod()
  }
}

export const openTelemetry = {
  init: appName => init(appName),
  getWebTracer: () => webTracer,
  withTelemetry,
}
