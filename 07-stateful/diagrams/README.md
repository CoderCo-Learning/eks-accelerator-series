# EP7 diagrams

Editable source for the Episode 7 teaching diagram. Open `ep7-stateful.drawio` in
[draw.io](https://app.diagrams.net) and export to PNG/SVG for slides.

Three pages:

1. **StatefulSet identity** - StatefulSet to pod to PVC to EBS, showing the pod name and its disk as a married pair.
2. **Headless DNS** - the app connecting by name through a headless Service to a specific pod, with the normal Service shown as the wrong choice.
3. **AZ-pinning** - the pod welded to its EBS volume in one AZ, unable to follow to another zone, plus the two honest fixes.

Colour key: blue is the controller and the app, green is the pod and the working path, amber is the storage, red is the failure and the wrong choice.
