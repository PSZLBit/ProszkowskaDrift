package pszlbit.drift.input

import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.util.AttributeSet
import android.view.MotionEvent
import android.view.View
import pszlbit.drift.R

/**
 * Placeholder digital touch scheme for the spike: a left/right steer pad and a
 * forward/reverse throttle pad, usable with two thumbs.
 *
 * D-14 is still TBD: the final scheme must be chosen by playtesting on the real
 * tablets, so treat this as a throwaway mapping, not a product decision.
 */
class InputPadView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
) : View(context, attrs) {
    private val tracker = InputTracker()
    private val zoneRects = HashMap<Zone, RectF>()

    private val fillPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.FILL
        color = resources.getColor(R.color.surface)
    }
    private val pressedPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.FILL
        color = resources.getColor(R.color.accent_pressed)
    }
    private val labelPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = resources.getColor(R.color.text_primary)
        textAlign = Paint.Align.CENTER
        textSize = sp(22f)
    }

    var onInputChanged: ((InputState) -> Unit)? = null

    override fun onSizeChanged(width: Int, height: Int, oldWidth: Int, oldHeight: Int) {
        super.onSizeChanged(width, height, oldWidth, oldHeight)
        zoneRects.clear()
        val gap = dp(6f)
        val margin = dp(4f)
        val midX = width / 2f
        val leftPad = RectF(margin, margin, midX - gap / 2f, height - margin)
        val rightPad = RectF(midX + gap / 2f, margin, width - margin, height - margin)
        zoneRects[Zone.LEFT] = RectF(leftPad.left, leftPad.top, leftPad.centerX(), leftPad.bottom)
        zoneRects[Zone.RIGHT] = RectF(leftPad.centerX(), leftPad.top, leftPad.right, leftPad.bottom)
        zoneRects[Zone.FORWARD] = RectF(rightPad.left, rightPad.top, rightPad.right, rightPad.centerY())
        zoneRects[Zone.REVERSE] = RectF(rightPad.left, rightPad.centerY(), rightPad.right, rightPad.bottom)
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        val radius = dp(10f)
        for ((zone, rect) in zoneRects) {
            canvas.drawRoundRect(rect, radius, radius, fillPaint)
            if (tracker.isPressed(zone)) {
                canvas.drawRoundRect(rect, radius, radius, pressedPaint)
            }
            val label = labelFor(zone)
            val textY = rect.centerY() - (labelPaint.descent() + labelPaint.ascent()) / 2f
            canvas.drawText(label, rect.centerX(), textY, labelPaint)
        }
    }

    override fun onTouchEvent(event: MotionEvent): Boolean {
        if (!isEnabled) return false
        when (event.actionMasked) {
            MotionEvent.ACTION_DOWN,
            MotionEvent.ACTION_POINTER_DOWN,
            -> {
                val index = event.actionIndex
                tracker.setPointer(event.getPointerId(index), zoneAt(event.getX(index), event.getY(index)))
            }

            MotionEvent.ACTION_MOVE -> {
                for (index in 0 until event.pointerCount) {
                    tracker.setPointer(
                        event.getPointerId(index),
                        zoneAt(event.getX(index), event.getY(index)),
                    )
                }
            }

            MotionEvent.ACTION_POINTER_UP -> tracker.releasePointer(event.getPointerId(event.actionIndex))
            MotionEvent.ACTION_UP,
            MotionEvent.ACTION_CANCEL,
            -> tracker.clear()
        }
        invalidate()
        onInputChanged?.invoke(tracker.state)
        return true
    }

    override fun setEnabled(enabled: Boolean) {
        super.setEnabled(enabled)
        if (!enabled) {
            tracker.clear()
            onInputChanged?.invoke(InputState.NEUTRAL)
            invalidate()
        }
    }

    private fun zoneAt(x: Float, y: Float): Zone? {
        for ((zone, rect) in zoneRects) {
            if (rect.contains(x, y)) return zone
        }
        return null
    }

    private fun labelFor(zone: Zone): String = when (zone) {
        Zone.LEFT -> context.getString(R.string.input_left)
        Zone.RIGHT -> context.getString(R.string.input_right)
        Zone.FORWARD -> context.getString(R.string.input_forward)
        Zone.REVERSE -> context.getString(R.string.input_reverse)
    }

    private fun dp(value: Float): Float = value * resources.displayMetrics.density

    private fun sp(value: Float): Float = value * resources.displayMetrics.scaledDensity
}
