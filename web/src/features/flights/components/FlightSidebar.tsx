import type { ReactNode } from 'react'
import type { FlightRecord } from '../types'

interface FlightSidebarProps {
  record: FlightRecord | null
  loading: boolean
  error: string | null
  onClose: () => void
}

function fmtTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '—' : d.toLocaleString()
}

function Row({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex justify-between gap-3 py-1 text-sm">
      <span className="text-gray-400">{label}</span>
      <span className="text-right text-gray-100">{value}</span>
    </div>
  )
}

export function FlightSidebar({
  record,
  loading,
  error,
  onClose,
}: FlightSidebarProps) {
  const f = record?.flight

  return (
    <div className="absolute z-20 flex flex-col bg-gray-900/95 text-white shadow-xl backdrop-blur
      inset-x-0 bottom-0 max-h-[40vh] rounded-t-xl
      md:inset-x-auto md:right-0 md:top-0 md:bottom-0 md:h-full md:w-80 md:max-h-none md:rounded-none">
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <div>
          <div className="text-lg font-semibold">
            {f?.callSign ?? (loading ? 'Loading…' : 'Flight')}
          </div>
          {f && (
            <div className="text-xs text-gray-400">
              {f.origin ?? '???'} → {f.destination ?? '???'}
            </div>
          )}
        </div>
        <button
          onClick={onClose}
          className="rounded-md px-2 py-1 text-gray-400 hover:bg-white/10 hover:text-white"
          aria-label="Close"
        >
          ✕
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3">
        {error && <div className="text-sm text-red-400">error: {error}</div>}
        {!error && loading && (
          <div className="text-sm text-gray-400">loading record…</div>
        )}
        {!error && !loading && !f && (
          <div className="text-sm text-gray-400">no record for this flight</div>
        )}

        {f && (
          <>
            {f.status && (
              <span className="mb-3 inline-block rounded bg-white/10 px-2 py-0.5 text-xs font-medium">
                {f.status}
              </span>
            )}
            <Row label="Registration" value={f.registration ?? '—'} />
            <Row label="Aircraft" value={f.aircraftType ?? '—'} />
            <Row label="Departed (actual)" value={fmtTime(f.actualDepartureTime)} />
            <Row label="Arrived (actual)" value={fmtTime(f.actualArrivalTime)} />
            <Row label="First seen" value={fmtTime(f.firstSeen)} />
            <Row label="Last seen" value={fmtTime(f.lastSeen)} />
            <Row label="Drop episodes" value={f.dropCount} />
            <Row label="Reactivations" value={f.reactivationCount} />

            <div className="mt-3 border-t border-white/10 pt-3">
              <Row label="Track points" value={record?.track.length ?? 0} />
            </div>

            <div className="mt-2 break-all text-[10px] text-gray-500">{f.gufi}</div>
          </>
        )}
      </div>
    </div>
  )
}
