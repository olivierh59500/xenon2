package com.olivierh.xenon2;

import android.content.Context;
import android.util.SparseBooleanArray;
import android.view.MotionEvent;

import com.olivierh.xenon2.ebitenmobileview.Ebitenmobileview;
import com.olivierh.xenon2.mobile.EbitenView;

/** Clears canceled Android contacts that Ebitengine 2.9 otherwise retains. */
final class XenonView extends EbitenView {
    private final SparseBooleanArray pointers = new SparseBooleanArray();

    XenonView(Context context) {
        super(context);
    }

    @Override
    public boolean onTouchEvent(MotionEvent event) {
        int action = event.getActionMasked();
        if (action == MotionEvent.ACTION_CANCEL) {
            releaseTouches();
            return true;
        }
        for (int i = 0; i < event.getPointerCount(); i++) {
            int id = event.getPointerId(i);
            if (i == event.getActionIndex()
                    && (action == MotionEvent.ACTION_UP || action == MotionEvent.ACTION_POINTER_UP)) {
                pointers.delete(id);
            } else {
                pointers.put(id, true);
            }
        }
        return super.onTouchEvent(event);
    }

    @Override
    public void suspendGame() {
        releaseTouches();
        super.suspendGame();
    }

    void releaseTouches() {
        for (int i = 0; i < pointers.size(); i++) {
            Ebitenmobileview.updateTouchesOnAndroid(MotionEvent.ACTION_UP, pointers.keyAt(i), 0, 0);
        }
        pointers.clear();
    }
}
