import { useEffect, useState } from 'react'
import type { FlightRecord } from '../types'
import { fetchFlightRecord } from '../api/flights'

export interface FlightRecordState {
  data: FlightRecord | null
  loading: boolean
  error: string | null
}

export function useFlightRecord(gufi: string | null): FlightRecordState {
  const [data, setData] = useState<FlightRecord | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!gufi) {
      setData(null)
      setError(null)
      setLoading(false)
      return
    }

    let cancelled = false
    const controller = new AbortController()
    setLoading(true)
    setError(null)

    fetchFlightRecord(gufi, controller.signal)
      .then((r) => {
        if (!cancelled) {
          setData(r)
          setLoading(false)
        }
      })
      .catch((e) => {
        const err = e as Error
        if (!cancelled && err.name !== 'AbortError') {
          setError(err.message)
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [gufi])

  return { data, loading, error }
}
