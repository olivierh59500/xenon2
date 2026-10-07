#!/bin/sh
# Exercise the actual native touch adapter against the pinned bridge behavior.
# The stubs model Ebitengine 2.9's ignored CANCEL and retained suspended touches;
# no Android runtime, emulator or additional testing library is required.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
java_home_path=${JAVA_HOME:-/opt/homebrew/opt/openjdk@17}
if [ ! -x "$java_home_path/bin/javac" ]; then
    echo "An existing JDK is required. Set JAVA_HOME." >&2
    exit 1
fi
test_root="$project_root/.local/android-touch-test"
mkdir -p "$test_root/src/android/content" "$test_root/src/android/util" \
    "$test_root/src/android/view" "$test_root/src/com/olivierh/xenon2/mobile" \
    "$test_root/src/com/olivierh/xenon2/ebitenmobileview" \
    "$test_root/src/com/olivierh/xenon2" "$test_root/classes"

cat > "$test_root/src/android/content/Context.java" <<'JAVA'
package android.content;
public class Context {}
JAVA
cat > "$test_root/src/android/util/SparseBooleanArray.java" <<'JAVA'
package android.util;
import java.util.ArrayList;
import java.util.TreeMap;
public class SparseBooleanArray {
    private final TreeMap<Integer, Boolean> values = new TreeMap<>();
    public void put(int key, boolean value) { values.put(key, value); }
    public void delete(int key) { values.remove(key); }
    public void clear() { values.clear(); }
    public int size() { return values.size(); }
    public int keyAt(int index) { return new ArrayList<>(values.keySet()).get(index); }
}
JAVA
cat > "$test_root/src/android/view/MotionEvent.java" <<'JAVA'
package android.view;
public class MotionEvent {
    public static final int ACTION_DOWN = 0, ACTION_UP = 1, ACTION_MOVE = 2,
            ACTION_CANCEL = 3, ACTION_POINTER_DOWN = 5, ACTION_POINTER_UP = 6;
    private final int action, index;
    private final int[] ids;
    public MotionEvent(int action, int index, int... ids) {
        this.action = action; this.index = index; this.ids = ids;
    }
    public int getActionMasked() { return action; }
    public int getActionIndex() { return index; }
    public int getPointerCount() { return ids.length; }
    public int getPointerId(int index) { return ids[index]; }
}
JAVA
cat > "$test_root/src/com/olivierh/xenon2/ebitenmobileview/Ebitenmobileview.java" <<'JAVA'
package com.olivierh.xenon2.ebitenmobileview;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
public class Ebitenmobileview {
    public static final Set<Long> held = new LinkedHashSet<>();
    public static final List<Long> releases = new ArrayList<>();
    public static void updateTouchesOnAndroid(long action, long id, long x, long y) {
        if (action == 0 || action == 5 || action == 2) { held.add(id); }
        if (action == 1 || action == 6) { held.remove(id); releases.add(id); }
        // Ebitengine 2.9 does not handle ACTION_CANCEL.
    }
}
JAVA
cat > "$test_root/src/com/olivierh/xenon2/mobile/EbitenView.java" <<'JAVA'
package com.olivierh.xenon2.mobile;
import android.content.Context;
import android.view.MotionEvent;
import com.olivierh.xenon2.ebitenmobileview.Ebitenmobileview;
public class EbitenView {
    public boolean suspended;
    public EbitenView(Context context) {}
    public boolean onTouchEvent(MotionEvent event) {
        for (int i = 0; i < event.getPointerCount(); i++) {
            int action = i == event.getActionIndex() ? event.getActionMasked() : MotionEvent.ACTION_MOVE;
            Ebitenmobileview.updateTouchesOnAndroid(action, event.getPointerId(i), 0, 0);
        }
        return true;
    }
    public void suspendGame() { suspended = true; }
}
JAVA
cat > "$test_root/src/com/olivierh/xenon2/XenonViewLifecycleTest.java" <<'JAVA'
package com.olivierh.xenon2;
import android.content.Context;
import android.view.MotionEvent;
import com.olivierh.xenon2.ebitenmobileview.Ebitenmobileview;
public class XenonViewLifecycleTest {
    private static void check(boolean success, String failure) {
        if (!success) { throw new AssertionError(failure); }
    }
    private static void threeFingers(XenonView view) {
        view.onTouchEvent(new MotionEvent(MotionEvent.ACTION_DOWN, 0, 3));
        view.onTouchEvent(new MotionEvent(MotionEvent.ACTION_POINTER_DOWN, 1, 3, 9));
        view.onTouchEvent(new MotionEvent(MotionEvent.ACTION_POINTER_DOWN, 2, 3, 9, 12));
        check(Ebitenmobileview.held.size() == 3, "three independent contacts must be held");
    }
    public static void main(String[] args) {
        XenonView view = new XenonView(new Context());
        threeFingers(view);
        view.onTouchEvent(new MotionEvent(MotionEvent.ACTION_CANCEL, 0, 3, 9, 12));
        check(Ebitenmobileview.held.isEmpty(), "CANCEL must release every contact");
        check(Ebitenmobileview.releases.size() == 3, "CANCEL must send three UP events");
        threeFingers(view);
        view.suspendGame();
        check(view.suspended, "suspension must reach Ebitengine");
        check(Ebitenmobileview.held.isEmpty(), "suspension must clear held controls");
        threeFingers(view);
        view.onTouchEvent(new MotionEvent(MotionEvent.ACTION_POINTER_UP, 1, 3, 9, 12));
        check(Ebitenmobileview.held.size() == 2 && !Ebitenmobileview.held.contains(9L), "normal UP must only release its pointer");
        int before = Ebitenmobileview.releases.size();
        view.releaseTouches();
        check(Ebitenmobileview.held.isEmpty(), "focus loss must clear remaining contacts");
        check(Ebitenmobileview.releases.size() == before + 2, "focus loss must release exactly the remaining pointers");
        view.releaseTouches();
        check(Ebitenmobileview.releases.size() == before + 2, "duplicate lifecycle callbacks must not repeat releases");
        System.out.println("Android touch cancellation and suspension checks passed.");
    }
}
JAVA

"$java_home_path/bin/javac" -d "$test_root/classes" \
    "$test_root/src/android/content/Context.java" \
    "$test_root/src/android/util/SparseBooleanArray.java" \
    "$test_root/src/android/view/MotionEvent.java" \
    "$test_root/src/com/olivierh/xenon2/ebitenmobileview/Ebitenmobileview.java" \
    "$test_root/src/com/olivierh/xenon2/mobile/EbitenView.java" \
    "$project_root/android/app/src/main/java/com/olivierh/xenon2/XenonView.java" \
    "$test_root/src/com/olivierh/xenon2/XenonViewLifecycleTest.java"
"$java_home_path/bin/java" -cp "$test_root/classes" com.olivierh.xenon2.XenonViewLifecycleTest
