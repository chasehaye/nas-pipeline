import { sendRequest } from '../../../lib/apiClient'
import type { Flight, FlightsResponse, FlightRecord } from '../types'


export async function fetchFlights(signal?: AbortSignal): Promise<Flight[]> {
  const data = await sendRequest<FlightsResponse>('/flights', { signal })
  return data.flights ?? []
}

export async function fetchFlightRecord(
  gufi: string,
  signal?: AbortSignal,
): Promise<FlightRecord> {
  return sendRequest<FlightRecord>(
    `/durable/flights/${encodeURIComponent(gufi)}`,
    { signal },
  )
}
