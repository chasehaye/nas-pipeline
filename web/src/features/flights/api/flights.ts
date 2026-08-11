import { sendRequest } from '../../../lib/apiClient'
import type { Flight, FlightsResponse } from '../types'


export async function fetchFlights(signal?: AbortSignal): Promise<Flight[]> {
  const data = await sendRequest<FlightsResponse>('/flights', { signal })
  return data.flights ?? []
}
