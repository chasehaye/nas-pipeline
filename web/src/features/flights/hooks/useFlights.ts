import { useEffect, useState } from 'react'
import type { Flight } from '../types'
import { fetchFlights } from '../api/flights'

export interface FlightsState {
  flights: Flight[]
  count: number
  error: string | null
  lastUpdated: number | null
}

export function useFlights(intervalMs = 4000): FlightsState {
  const [flights, setFlights] = useState<Flight[]>([])
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    const controller = new AbortController()

    async function poll() {
      try {
        const next = await fetchFlights(controller.signal)
        if (!cancelled) {
          setFlights(next)
          setError(null)
          setLastUpdated(Date.now())
        }
      } catch (e) {
        const err = e as Error
        if (!cancelled && err.name !== 'AbortError') {
          setError(err.message)
        }
      }
    }

    poll()
    const id = setInterval(poll, intervalMs)
    return () => {
      cancelled = true
      controller.abort()
      clearInterval(id)
    }
  }, [intervalMs])

  return { flights, count: flights.length, error, lastUpdated }
}
